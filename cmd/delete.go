// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <nome1> [nome2...]",
	Short: "Remove uma ou mais tags e todo o seu índice de rastreamento",
	Args:  cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		// Retirada a restrição de len(args) != 0 para continuar sugerindo tags
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		// Transação única para deletar múltiplas tags em lote
		err = db.Update(func(tx *bbolt.Tx) error {
			projBucket := tx.Bucket([]byte(storage.BucketTags))
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))

			for _, tagName := range args {
				// Deleta a referência da tag
				if projBucket.Get([]byte(tagName)) != nil {
					if err := projBucket.Delete([]byte(tagName)); err != nil {
						return fmt.Errorf("falha ao remover referência da tag '%s': %w", tagName, err)
					}
				}

				// Deleta o bucket de arquivos rastreados associados à tag
				if filesBucket.Bucket([]byte(tagName)) != nil {
					if err := filesBucket.DeleteBucket([]byte(tagName)); err != nil {
						return fmt.Errorf("falha ao remover rastreamento da tag '%s': %w", tagName, err)
					}
				}
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao deletar tags: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Tags deletadas com sucesso: %v\n", args)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
