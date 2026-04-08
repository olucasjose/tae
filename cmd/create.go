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
	Use:   "create <nome1> [nome2...]",
	Short: "Cria uma ou mais tags de rastreamento no banco de dados",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		err = db.Update(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(storage.BucketTags))
			
			// Validação Fail-Fast para evitar estado inconsistente
			for _, tagName := range args {
				if b.Get([]byte(tagName)) != nil {
					return fmt.Errorf("a tag '%s' já existe. Operação abortada", tagName)
				}
			}
			
			// Persistência
			for _, tagName := range args {
				if err := b.Put([]byte(tagName), []byte("{}")); err != nil {
					return fmt.Errorf("erro ao gravar tag '%s': %w", tagName, err)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro na transação: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Tag(s) criada(s) com sucesso: %v\n", args)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
