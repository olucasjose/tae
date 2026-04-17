// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"

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
	Run: func(cmd *cobra.Command, args []string) {
		if err := storage.DeleteTags(args); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao deletar tags: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Tags deletadas com sucesso: %v\n", args)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
