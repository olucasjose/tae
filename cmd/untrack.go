// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"

	"tae/internal/storage"
	"tae/internal/fs"

	"github.com/spf13/cobra"
)

var untrackCmd = &cobra.Command{
	Use:   "untrack <arquivo1> [arquivo2...] <nome da tag>",
	Short: "Remove um ou mais arquivos/diretórios do monitoramento de uma tag",
	Args:  cobra.MinimumNArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveDefault
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tagName := args[len(args)-1]
		targets := args[:len(args)-1]

		resolvedTargets, err := resolveTagPaths(tagName, targets)
		if err != nil {
			return fmt.Errorf("erro de resolução: %w", err)
		}

		for i, target := range resolvedTargets {
			if err := storage.UntrackPath(tagName, target); err != nil {
				// Falha rápida se houver erro ao destrackear
				return fmt.Errorf("erro ao remover alvo '%s': %w", targets[i], err)
			}
			fmt.Printf("Alvo '%s' removido da tag '%s'.\n", targets[i], tagName)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(untrackCmd)
}
