// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"tae/internal/storage"
	"tae/internal/fs"

	"github.com/spf13/cobra"
)

var (
	pruneAll   bool
	pruneList  bool
	pruneForce bool
	pruneQuiet bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune [nome1] [nome2...]",
	Short: "Remove do banco os alvos (rastreados ou na denylist) que não existem mais no disco",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && !pruneAll {
			return fmt.Errorf("informe pelo menos uma tag ou use a flag --all (-a) para atuar em todas")
		}

		if pruneQuiet && !pruneForce && !pruneList {
			return fmt.Errorf("a flag --quiet (-q) exige o uso de --force (-f) para evitar que o terminal aguarde confirmação invisivelmente")
		}

		var targetTags []string
		if pruneAll {
			tagsMeta, _ := storage.GetAllTagsWithMeta()
			for t := range tagsMeta {
				targetTags = append(targetTags, t)
			}
		} else {
			targetTags = args
		}

		ghostsFilesByTag := make(map[string][]string)
		ghostsIgnoredByTag := make(map[string][]string)
		totalGhosts := 0

		// Etapa 1: Escaneamento em memória (sem bloquear banco)
		for _, tagName := range targetTags {
			rawFiles, rawIgnored, err := storage.GetTagRawKeys(tagName)
			if err != nil {
				continue
			}

			resolvedFiles, errF := restorePathsForDisk(tagName, rawFiles)
			resolvedIgnored, errI := restorePathsForDisk(tagName, rawIgnored)

			if errF == nil {
				for i, p := range resolvedFiles {
					if _, err := os.Stat(p); os.IsNotExist(err) {
						ghostsFilesByTag[tagName] = append(ghostsFilesByTag[tagName], rawFiles[i])
						totalGhosts++
					}
				}
			}
			if errI == nil {
				for i, p := range resolvedIgnored {
					if _, err := os.Stat(p); os.IsNotExist(err) {
						ghostsIgnoredByTag[tagName] = append(ghostsIgnoredByTag[tagName], rawIgnored[i])
						totalGhosts++
					}
				}
			}

			if !pruneAll && !pruneQuiet && len(ghostsFilesByTag[tagName]) == 0 && len(ghostsIgnoredByTag[tagName]) == 0 {
				fmt.Printf("Verificando tag '%s'...\n", tagName)
			}
		}

		// Etapa 2: Exibição e Interação
		if totalGhosts == 0 {
			if !pruneQuiet {
				fmt.Println("Nenhum arquivo fantasma encontrado. Os índices estão atualizados.")
			}
			return nil
		}

		if !pruneQuiet {
			allTagsMap := make(map[string]bool)
			for t := range ghostsFilesByTag {
				allTagsMap[t] = true
			}
			for t := range ghostsIgnoredByTag {
				allTagsMap[t] = true
			}

			for tagName := range allTagsMap {
				fLen := len(ghostsFilesByTag[tagName])
				iLen := len(ghostsIgnoredByTag[tagName])
				if fLen > 0 || iLen > 0 {
					fmt.Printf("Tag '%s': %d arquivo(s) fantasma(s) detectado(s).\n", tagName, fLen+iLen)
					for _, k := range ghostsFilesByTag[tagName] {
						fmt.Printf("  - %s [Rastreado]\n", k)
					}
					for _, k := range ghostsIgnoredByTag[tagName] {
						fmt.Printf("  - %s [Denylist]\n", k)
					}
				}
			}
		}

		if pruneList {
			if !pruneQuiet {
				fmt.Printf("\nTotal detectado: %d arquivo(s).\n", totalGhosts)
			}
			return nil
		}

		if !pruneForce {
			fmt.Printf("\nDeseja remover %d arquivo(s) fantasma(s) permanentemente dos índices? [s/N]: ", totalGhosts)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "s" && response != "y" {
				fmt.Println("Operação cancelada.")
				return nil
			}
		}

		// Etapa 3: Execução Destrutiva
		for tagName := range ghostsFilesByTag {
			if err := storage.RemoveKeysFromTag(tagName, ghostsFilesByTag[tagName], ghostsIgnoredByTag[tagName]); err != nil {
				fmt.Printf("Erro ao limpar tag '%s': %v\n", tagName, err) // Aviso contínuo para não travar o loop
			}
		}

		for tagName := range ghostsIgnoredByTag {
			if _, exists := ghostsFilesByTag[tagName]; !exists {
				if err := storage.RemoveKeysFromTag(tagName, nil, ghostsIgnoredByTag[tagName]); err != nil {
					fmt.Printf("Erro ao limpar tag '%s': %v\n", tagName, err)
				}
			}
		}

		if !pruneQuiet {
			fmt.Printf("Sucesso! %d arquivo(s) fantasma(s) removido(s) do banco.\n", totalGhosts)
		}
		return nil
	},
}

func init() {
	pruneCmd.Flags().BoolVarP(&pruneAll, "all", "a", false, "Aplica a verificação em todas as tags cadastradas")
	pruneCmd.Flags().BoolVarP(&pruneList, "list", "l", false, "Apenas lista os arquivos fantasmas sem removê-los")
	pruneCmd.Flags().BoolVarP(&pruneForce, "force", "f", false, "Força a exclusão sem solicitar confirmação do usuário")
	pruneCmd.Flags().BoolVarP(&pruneQuiet, "quiet", "q", false, "Oculta a saída no terminal (requer -f ou -l)")
	rootCmd.AddCommand(pruneCmd)
}
