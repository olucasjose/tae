// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <tag_antiga> <tag_nova>",
	Short: "Renomeia uma tag existente e transfere todo o seu rastreamento",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Sugere nomes apenas para o primeiro argumento (a tag existente)
		if len(args) == 0 {
			tags, _ := storage.GetAllTags()
			return tags, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		oldTag := args[0]
		newTag := args[1]

		if err := storage.RenameTag(oldTag, newTag); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao renomear tag: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Tag '%s' renomeada para '%s' com sucesso.\n", oldTag, newTag)
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
