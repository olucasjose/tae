// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"

	"tae/internal/storage"
	"tae/internal/fs"

	"github.com/spf13/cobra"
)

var ignoreRemove bool

var ignoreCmd = &cobra.Command{
	Use:   "ignore <arquivo1> [arquivo2...] <nome da tag>",
	Short: "Gerencia os arquivos na denylist da tag (Exclusion Index)",
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

		// Bifurcação de estado via Flag
		if ignoreRemove {
			if err := storage.UnignorePaths(tagName, resolvedTargets); err != nil {
				return fmt.Errorf("erro ao remover alvos da denylist: %w", err)
			}
			fmt.Printf("%d alvo(s) removido(s) da denylist da tag '%s'.\n", len(targets), tagName)
			return nil
		}

		if err := storage.IgnorePaths(tagName, resolvedTargets); err != nil {
			return fmt.Errorf("erro ao ignorar alvos: %w", err)
		}

		fmt.Printf("%d alvo(s) adicionado(s) à denylist da tag '%s'.\n", len(targets), tagName)
		return nil
	},
}

func init() {
	ignoreCmd.Flags().BoolVarP(&ignoreRemove, "remove", "r", false, "Remove os alvos da denylist (restaura o rastreamento por herança)")
	rootCmd.AddCommand(ignoreCmd)
}
