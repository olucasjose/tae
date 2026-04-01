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
			
			// Checa se já existe para não sobrescrever acidentalmente
			if b.Get([]byte(projectName)) != nil {
				return fmt.Errorf("o projeto '%s' já existe", projectName)
			}
			
			// Salvamos o nome como chave. O valor pode ser um JSON de configs futuras.
			return b.Put([]byte(projectName), []byte("{}"))
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro na transação: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Projeto '%s' criado e pronto para rastrear arquivos.\n", projectName)
	},
}

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectCreateCmd)
}
