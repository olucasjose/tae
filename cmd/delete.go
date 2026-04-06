package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var deleteCmd = &cobra.Command{
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
	rootCmd.AddCommand(deleteCmd)
}
