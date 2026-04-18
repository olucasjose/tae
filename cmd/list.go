// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"tae/internal/render"
	"tae/internal/storage"
	"tae/internal/fs"

	"github.com/spf13/cobra"
)

var (
	listTree     bool
	listDepth    int
	listIgnore   string
	listAbsolute bool
	listExpand   bool
	listIgnored  bool
	listDetails  bool
	listGroup    bool
)

var listCmd = &cobra.Command{
	Use:   "list [nome da tag]",
	Short: "Lista todas as tags ou os arquivos rastreados de uma tag específica",
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			tagsMeta, err := storage.GetAllTagsWithMeta()
			if err != nil {
				return fmt.Errorf("erro ao carregar tags: %w", err)
			}

			if listGroup {
				groups := make(map[string][]string)

				for tag, meta := range tagsMeta {
					repo := "No repo"
					if meta.Type == storage.TagTypeGit {
						repo = meta.RepoName
						if repo == "" {
							repo = meta.RepoID
						}
					}
					groups[repo] = append(groups[repo], tag)
				}

				var repos []string
				for r := range groups {
					if r != "No repo" {
						repos = append(repos, r)
					}
				}
				sort.Strings(repos)

				if tags, ok := groups["No repo"]; ok {
					fmt.Println("No repo:")
					sort.Strings(tags)
					for _, t := range tags {
						fmt.Printf("\t%s\n", t)
					}
					if len(repos) > 0 {
						fmt.Println()
					}
				}

				for i, r := range repos {
					fmt.Printf("\033[33m%s:\033[0m\n", r)
					tags := groups[r]
					sort.Strings(tags)
					for _, t := range tags {
						fmt.Printf("\t%s\n", t)
					}
					if i < len(repos)-1 {
						fmt.Println()
					}
				}
				return nil
			}

			if !listDetails {
				fmt.Println("Tags cadastradas:")
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			if listDetails {
				fmt.Fprintln(w, "TAG\tTIPO\tREPOSITÓRIO")
			}

			var tagNames []string
			for t := range tagsMeta {
				tagNames = append(tagNames, t)
			}
			sort.Strings(tagNames)

			for _, tagName := range tagNames {
				meta := tagsMeta[tagName]
				if listDetails {
					if meta.Type == storage.TagTypeGit {
						repoName := meta.RepoName
						if repoName == "" {
							repoName = meta.RepoID
						}
						fmt.Fprintf(w, "%s\tGit\t%s\n", tagName, repoName)
					} else {
						fmt.Fprintf(w, "%s\tLocal\t\n", tagName)
					}
				} else {
					fmt.Printf("  - %s\n", tagName)
				}
			}

			if listDetails {
				w.Flush()
			}
			return nil
		}

		tagName := args[0]

		if listIgnored {
			ignoredMap, err := storage.GetIgnoredPaths(tagName)
			if err != nil {
				return fmt.Errorf("erro ao ler Exclusion Index: %w", err)
			}

			if len(ignoredMap) == 0 {
				fmt.Printf("A denylist da tag '%s' está vazia.\n", tagName)
				return nil
			}

			fmt.Printf("Exclusion Index (Denylist) da tag '%s':\n", tagName)
			for path := range ignoredMap {
				fmt.Printf("  - %s\n", path)
			}
			return nil
		}

		files, err := storage.GetFilesByTag(tagName)
		if err != nil {
			return fmt.Errorf("erro ao consultar arquivos: %w", err)
		}

		if len(files) == 0 {
			fmt.Printf("Alvos rastreados na tag '%s':\n  (Vazio ou tag não inicializada)\n", tagName)
			return nil
		}

		resolvedFiles, err := restorePathsForDisk(tagName, files)
		if err != nil {
			return fmt.Errorf("erro de escopo estrutural: %w", err)
		}
		files = resolvedFiles

		if listExpand {
			ignoredMap, _ := storage.GetIgnoredPaths(tagName)

			restoredIgnored := make(map[string]bool)
			var igPaths []string
			for p := range ignoredMap { igPaths = append(igPaths, p) }
			if resIgPaths, err := restorePathsForDisk(tagName, igPaths); err == nil {
				for _, p := range resIgPaths { restoredIgnored[p] = true }
			}

			files = expandPathsToFiles(files, restoredIgnored)
		}

		fmt.Printf("Alvos rastreados na tag '%s':\n", tagName)

		if listAbsolute {
			for _, f := range files {
				fmt.Printf("  - %s\n", f)
			}
			return nil
		}

		basePrefix := render.GetCommonPrefix(files)
		var ignorePatterns []string
		if listIgnore != "" {
			ignorePatterns = strings.Split(listIgnore, "|")
		}

		fmt.Printf("[Raiz Comum: %s]\n\n", basePrefix)

		if listTree {
			rootNode := render.BuildVisualTree(files, basePrefix)
			render.PrintTree(rootNode, "", 0, listDepth, ignorePatterns)
		} else {
			for _, f := range files {
				relPath := strings.TrimPrefix(f, basePrefix)
				relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
				if relPath == "" {
					relPath = filepath.Base(f)
				}
				fmt.Printf("  - %s\n", relPath)
			}
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listTree, "tree", "t", false, "Exibe os caminhos em formato de árvore")
	listCmd.Flags().IntVarP(&listDepth, "level", "L", 0, "Profundidade máxima da árvore (0 = infinito)")
	listCmd.Flags().StringVarP(&listIgnore, "ignore", "I", "", "Padrões para ignorar na exibição (ex: \"node_modules|*.go\")")
	listCmd.Flags().BoolVarP(&listAbsolute, "absolute", "A", false, "Exibe os caminhos absolutos originais sem truncar")
	listCmd.Flags().BoolVarP(&listExpand, "expand", "e", false, "Expande diretórios lendo o disco físico antes de listar")
	listCmd.Flags().BoolVarP(&listIgnored, "ignored", "i", false, "Exibe apenas os arquivos na denylist permanente da tag")
	listCmd.Flags().BoolVarP(&listDetails, "details", "d", false, "Exibe os metadados das tags em colunas, indicando se são Local ou Git")
	listCmd.Flags().BoolVarP(&listGroup, "group", "g", false, "Agrupa a exibição de tags por repositório (com suporte a cores)")
	rootCmd.AddCommand(listCmd)
}
