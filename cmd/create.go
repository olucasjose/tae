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

var createCmd = &cobra.Command{
	Use:   "create <nome>",
	Short: "Cria uma nova tag de rastreamento no banco de dados",
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

func init() {
	rootCmd.AddCommand(createCmd)
}
