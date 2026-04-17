// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <nome1> [nome2...]",
	Short: "Remove uma ou mais tags e todo o seu índice de rastreamento",
	Args:  cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	// Gina: Assinatura alterada de Run para RunE
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := storage.DeleteTags(args); err != nil {
			// Gina: Retorna o erro em vez de imprimir e chamar os.Exit(1)
			return fmt.Errorf("falha ao deletar tags: %w", err)
		}

		fmt.Printf("Tags deletadas com sucesso: %v\n", args)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
