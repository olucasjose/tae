// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tae/internal/render"
	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var (
	listTree     bool
	listDepth    int
	listIgnore   string
	listAbsolute bool
	listExpand   bool
	listIgnored  bool
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
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Caso sem argumentos: Lista todas as tags isoladamente
		if len(args) == 0 {
			db, err := storage.Open()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Tags cadastradas:")
			db.View(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte(storage.BucketTags))
				return b.ForEach(func(k, v []byte) error {
					fmt.Printf("  - %s\n", k)
					return nil
				})
			})
			db.Close() // Fecha e libera o lock do arquivo
			return
		}

		tagName := args[0]

		// 2. Interceptação da Denylist isolada
		if listIgnored {
			ignoredMap, err := storage.GetIgnoredPaths(tagName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao ler Exclusion Index: %v\n", err)
				os.Exit(1)
			}

			if len(ignoredMap) == 0 {
				fmt.Printf("A denylist da tag '%s' está vazia.\n", tagName)
				return
			}

			fmt.Printf("Exclusion Index (Denylist) da tag '%s':\n", tagName)
			for path := range ignoredMap {
				fmt.Printf("  - %s\n", path)
			}
			return
		}

		// 3. Busca principal isolada (Abre e fecha o banco rapidamente numa closure)
		var files []string
		err := func() error {
			db, err := storage.Open()
			if err != nil {
				return err
			}
			defer db.Close() // Fecha o banco no fim deste bloco

			return db.View(func(tx *bbolt.Tx) error {
				filesBucket := tx.Bucket([]byte(storage.BucketFiles))
				if filesBucket == nil {
					return nil
				}
				projFiles := filesBucket.Bucket([]byte(tagName))
				if projFiles == nil {
					return nil
				}

				return projFiles.ForEach(func(k, v []byte) error {
					files = append(files, string(k))
					return nil
				})
			})
		}()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao consultar arquivos: %v\n", err)
			os.Exit(1)
		}

		if len(files) == 0 {
			fmt.Printf("Alvos rastreados na tag '%s':\n  (Vazio ou tag não inicializada)\n", tagName)
			return
		}

		// 4. Expansão (Agora pode chamar o banco em O(1) sem colidir locks)
		if listExpand {
			ignoredMap, _ := storage.GetIgnoredPaths(tagName)
			files = expandPathsToFiles(files, ignoredMap)
		}

		fmt.Printf("Alvos rastreados na tag '%s':\n", tagName)

		// Exibição Absoluta (Legado)
		if listAbsolute {
			for _, f := range files {
				fmt.Printf("  - %s\n", f)
			}
			return
		}

		// Preparação da engine visual e caminhos relativos
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
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listTree, "tree", "t", false, "Exibe os caminhos em formato de árvore")
	listCmd.Flags().IntVarP(&listDepth, "level", "L", 0, "Profundidade máxima da árvore (0 = infinito)")
	listCmd.Flags().StringVarP(&listIgnore, "ignore", "I", "", "Padrões para ignorar na exibição (ex: \"node_modules|*.go\")")
	listCmd.Flags().BoolVarP(&listAbsolute, "absolute", "A", false, "Exibe os caminhos absolutos originais sem truncar")
	listCmd.Flags().BoolVarP(&listExpand, "expand", "e", false, "Expande diretórios lendo o disco físico antes de listar")
	listCmd.Flags().BoolVarP(&listIgnored, "ignored", "i", false, "Exibe apenas os arquivos na denylist permanente da tag")
	rootCmd.AddCommand(listCmd)
}
