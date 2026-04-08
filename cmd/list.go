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

var listCmd = &cobra.Command{
	Use:   "list [nome da tag]",
	Short: "Lista todas as tags ou os arquivos rastreados de uma tag específica",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp // Sem sugestão após a primeira tag
		}
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp // Retorna tags sem arquivos do disco
	},
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

func init() {
	rootCmd.AddCommand(listCmd)
}
