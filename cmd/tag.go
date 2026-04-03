package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Gerencia as tags monitoradas pelo Tae",
}

var tagCreateCmd = &cobra.Command{
	Use:   "create <nome>",
	Short: "Cria uma nova tag no banco de dados",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[0]

		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		err = db.Update(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(storage.BucketTags))
			if b.Get([]byte(tagName)) != nil {
				return fmt.Errorf("a tag '%s' já existe", tagName)
			}
			return b.Put([]byte(tagName), []byte("{}"))
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro na transação: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Tag '%s' criada e pronta para rastrear arquivos.\n", tagName)
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list [nome da tag]",
	Short: "Lista todas as tags ou os caminhos rastreados de uma tag específica",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		if len(args) == 0 {
			fmt.Println("Tags cadastradas:")
			db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(storage.BucketTags))
				return b.ForEach(func(k, v []byte) error {
					fmt.Printf("  - %s\n", k)
					return nil
				})
			})
			return
		}

		tagName := args[0]
		fmt.Printf("Alvos rastreados na tag '%s':\n", tagName)
		db.View(func(tx *bbolt.Tx) error {
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			projFiles := filesBucket.Bucket([]byte(tagName))
			
			if projFiles == nil {
				fmt.Println("  (Nenhum arquivo rastreado ou tag não inicializada)")
				return nil
			}

			count := 0
			projFiles.ForEach(func(k, v []byte) error {
				fmt.Printf("  - %s\n", k)
				count++
				return nil
			})
			
			if count == 0 {
				fmt.Println("  (Vazio)")
			}
			return nil
		})
	},
}

// 1. DEFINIÇÃO do comando delete
var tagDeleteCmd = &cobra.Command{
	Use:   "delete <nome>",
	Short: "Remove uma tag e todo o seu índice de rastreamento",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[0]

		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		err = db.Update(func(tx *bbolt.Tx) error {
			projBucket := tx.Bucket([]byte(storage.BucketTags))
			if err := projBucket.Delete([]byte(tagName)); err != nil {
				return err
			}

			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			if filesBucket.Bucket([]byte(tagName)) != nil {
				if err := filesBucket.DeleteBucket([]byte(tagName)); err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao deletar tag: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Tag '%s' e seus rastreamentos foram deletados com sucesso.\n", tagName)
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.AddCommand(tagCreateCmd)
	tagCmd.AddCommand(tagListCmd)
	// 2. REGISTRO do comando delete
	tagCmd.AddCommand(tagDeleteCmd) 
}
