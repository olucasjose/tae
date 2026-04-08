// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var ignorePattern string

var trackCmd = &cobra.Command{
	Use:   "track <arquivo1> [arquivo2...] <nome da tag>",
	Short: "Adiciona um ou mais arquivos/diretórios ao monitoramento de uma tag",
	Args:  cobra.MinimumNArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveDefault
	},
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[len(args)-1]
		targets := args[:len(args)-1]

		var ignorePatterns []string
		if ignorePattern != "" {
			ignorePatterns = strings.Split(ignorePattern, "|")
		}

		for _, target := range targets {
			if shouldIgnore(target, ignorePatterns) {
				fmt.Printf("Ignorando alvo: %s\n", target)
				continue
			}

			if _, err := os.Stat(target); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Aviso: O alvo '%s' não existe no disco. Ignorando.\n", target)
				continue
			}

			if err := storage.TrackPath(tagName, target); err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao rastrear '%s': %v\n", target, err)
			} else {
				fmt.Printf("Alvo '%s' rastreado com sucesso na tag '%s'.\n", target, tagName)
			}
		}
	},
}

// shouldIgnore avalia se o target bate com algum padrão.
// O filepath.Match não cruza delimitadores de caminho com '*', 
// o que atende à regra de ignorar apenas arquivos rasos da pasta atual.
func shouldIgnore(target string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, p := range patterns {
		p = strings.TrimSpace(p)
		matched, err := filepath.Match(p, target)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func init() {
	trackCmd.Flags().StringVarP(&ignorePattern, "ignore", "i", "", "Padrões para ignorar arquivos apenas na pasta atual (ex: \"node_modules|*.kt\")")
	rootCmd.AddCommand(trackCmd)
}
