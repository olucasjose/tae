// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"strings"

	"os"
	"tae/internal/filter"
	"tae/internal/render"
	"tae/internal/storage"
	"tae/internal/vcs"

	"github.com/spf13/cobra"
)

var (
	gitListTree     bool
	gitListDepth    int
	gitListIgnore   string
	gitListIgnored  bool
	gitListNoIgnore bool
)

var gitListCmd = &cobra.Command{
	Use:   "list [commit]",
	Short: "Lista arquivos de um commit ou a denylist do repositório atual",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if gitListIgnored {
			repoID := vcs.GetRepoID()
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				return fmt.Errorf("erro ao ler a denylist do repositório: %w", err)
			}

			if len(ignoredMap) == 0 {
				fmt.Println("A denylist do repositório atual está vazia.")
				return nil
			}

			fmt.Println("Exclusion Index (Denylist) do repositório atual:")
			for path := range ignoredMap {
				fmt.Printf("  - %s\n", path)
			}
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("informe um <commit> para listar ou use a flag --ignored (-i) para ver a denylist")
		}

		commit := args[0]
		rawFiles, err := vcs.ListTree(commit)
		if err != nil {
			return err
		}

		if len(rawFiles) == 0 {
			fmt.Println("Nenhum arquivo encontrado neste commit.")
			return nil
		}

		var files []string
		if !gitListNoIgnore {
			repoID := vcs.GetRepoID()
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				fmt.Printf("Aviso: Falha ao carregar denylist do repositório: %v\n", err)
			}

			for _, f := range rawFiles {
				if !filter.IsPathIgnoredByMap(f, ignoredMap) {
					files = append(files, f)
				}
			}
		} else {
			files = rawFiles
		}

		if len(files) == 0 {
			fmt.Println("Todos os arquivos deste commit foram retidos pela denylist.")
			return nil
		}

		var ignorePatterns []string
		if gitListIgnore != "" {
			ignorePatterns = strings.Split(gitListIgnore, "|")
		}

		if gitListTree {
			rootNode := render.BuildVisualTree(files, "")
			render.PrintTree(os.Stdout, rootNode, "", 0, gitListDepth, ignorePatterns)
		} else {
			for _, f := range files {
				if !filter.MatchPattern(f, ignorePatterns) {
					fmt.Printf("  - %s\n", f)
				}
			}
		}
		return nil
	},
}

func init() {
	gitListCmd.Flags().BoolVarP(&gitListTree, "tree", "t", false, "Exibe os caminhos em formato de árvore")
	gitListCmd.Flags().IntVarP(&gitListDepth, "level", "L", 0, "Profundidade máxima da árvore (0 = infinito)")
	gitListCmd.Flags().StringVarP(&gitListIgnore, "ignore", "I", "", "Padrões para ignorar na exibição (ex: \"*.go\")")
	gitListCmd.Flags().BoolVarP(&gitListIgnored, "ignored", "i", false, "Exibe os arquivos na denylist do repositório")
	gitListCmd.Flags().BoolVar(&gitListNoIgnore, "no-ignore", false, "Ignora a denylist do repositório e exibe todos os arquivos")
	gitCmd.AddCommand(gitListCmd)
}
