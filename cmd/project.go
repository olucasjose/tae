package cmd

import (
	"fmt"
	"os"

	"spycode/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Gerencia os projetos monitorados pelo Spycode",
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <nome>",
	Short: "Cria um novo projeto no banco de dados",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]

		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		err = db.Update(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(storage.BucketProjects))
			if b.Get([]byte(projectName)) != nil {
				return fmt.Errorf("o projeto '%s' já existe", projectName)
			}
			return b.Put([]byte(projectName), []byte("{}"))
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro na transação: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Projeto '%s' criado e pronto para rastrear arquivos.\n", projectName)
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list [nome do projeto]",
	Short: "Lista todos os projetos ou os caminhos rastreados de um projeto específico",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		if len(args) == 0 {
			fmt.Println("Projetos cadastrados:")
			db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(storage.BucketProjects))
				return b.ForEach(func(k, v []byte) error {
					fmt.Printf("  - %s\n", k)
					return nil
				})
			})
			return
		}

		projectName := args[0]
		fmt.Printf("Alvos rastreados no projeto '%s':\n", projectName)
		db.View(func(tx *bbolt.Tx) error {
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			projFiles := filesBucket.Bucket([]byte(projectName))
			
			if projFiles == nil {
				fmt.Println("  (Nenhum arquivo rastreado ou projeto não inicializado)")
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

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
}
