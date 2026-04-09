// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var (
	pruneAll   bool
	pruneList  bool
	pruneForce bool
	pruneQuiet bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune [nome1] [nome2...]",
	Short: "Remove do rastreamento os arquivos que não existem mais no disco",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		tags, _ := storage.GetAllTags()
		return tags, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && !pruneAll {
			fmt.Fprintln(os.Stderr, "Erro: Informe pelo menos uma tag ou use a flag --all (-a) para atuar em todas.")
			os.Exit(1)
		}

		if pruneQuiet && !pruneForce && !pruneList {
			fmt.Fprintln(os.Stderr, "Erro: A flag --quiet (-q) exige o uso de --force (-f) para evitar que o terminal aguarde confirmação invisivelmente.")
			os.Exit(1)
		}

		db, err := storage.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao conectar no banco: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		ghostsByTag := make(map[string][][]byte)
		totalGhosts := 0

		// Etapa 1: Escaneamento (View transacional - Não bloqueia o banco)
		err = db.View(func(tx *bbolt.Tx) error {
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			if filesBucket == nil {
				return nil
			}

			var targetTags []string
			if pruneAll {
				tagsBucket := tx.Bucket([]byte(storage.BucketTags))
				if tagsBucket != nil {
					_ = tagsBucket.ForEach(func(k, v []byte) error {
						targetTags = append(targetTags, string(k))
						return nil
					})
				}
			} else {
				targetTags = args
			}

			for _, tagName := range targetTags {
				projFiles := filesBucket.Bucket([]byte(tagName))
				if projFiles == nil {
					if !pruneAll && !pruneQuiet {
						fmt.Printf("Aviso: Tag '%s' não encontrada ou sem arquivos rastreados.\n", tagName)
					}
					continue
				}

				_ = projFiles.ForEach(func(k, v []byte) error {
					path := string(k)
					// Identificação Fail-Fast
					if _, err := os.Stat(path); os.IsNotExist(err) {
						ghostsByTag[tagName] = append(ghostsByTag[tagName], k)
						totalGhosts++
					}
					return nil
				})
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro durante a leitura do banco de dados: %v\n", err)
			os.Exit(1)
		}

		// Etapa 2: Exibição e Interação
		if totalGhosts == 0 {
			if !pruneQuiet {
				fmt.Println("Nenhum arquivo fantasma encontrado. O rastreamento está atualizado.")
			}
			return
		}

		if !pruneQuiet {
			for tagName, keys := range ghostsByTag {
				if len(keys) > 0 {
					fmt.Printf("Tag '%s': %d arquivo(s) fantasma(s) detectado(s).\n", tagName, len(keys))
					for _, k := range keys {
						fmt.Printf("  - %s\n", string(k))
					}
				}
			}
		}

		if pruneList {
			if !pruneQuiet {
				fmt.Printf("\nTotal detectado: %d arquivo(s).\n", totalGhosts)
			}
			return
		}

		if !pruneForce {
			fmt.Printf("\nDeseja remover %d arquivo(s) fantasma(s) permanentemente do índice? [s/N]: ", totalGhosts)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "s" && response != "y" {
				fmt.Println("Operação cancelada.")
				return
			}
		}

		// Etapa 3: Execução Destrutiva
		err = db.Update(func(tx *bbolt.Tx) error {
			filesBucket := tx.Bucket([]byte(storage.BucketFiles))
			if filesBucket == nil {
				return nil
			}

			for tagName, keys := range ghostsByTag {
				if len(keys) == 0 {
					continue
				}
				projFiles := filesBucket.Bucket([]byte(tagName))
				if projFiles == nil {
					continue // Condição de corrida: apagaram a tag durante o prompt
				}

				for _, k := range keys {
					if err := projFiles.Delete(k); err != nil {
						return fmt.Errorf("falha ao remover chave interna '%s': %w", string(k), err)
					}
				}
			}
			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro fatal na transação de deleção: %v\n", err)
			os.Exit(1)
		}

		if !pruneQuiet {
			fmt.Printf("Sucesso! %d arquivo(s) fantasma(s) removido(s).\n", totalGhosts)
		}
	},
}

func init() {
	pruneCmd.Flags().BoolVarP(&pruneAll, "all", "a", false, "Aplica a verificação em todas as tags cadastradas")
	pruneCmd.Flags().BoolVarP(&pruneList, "list", "l", false, "Apenas lista os arquivos fantasmas sem removê-los")
	pruneCmd.Flags().BoolVarP(&pruneForce, "force", "f", false, "Força a exclusão sem solicitar confirmação do usuário")
	pruneCmd.Flags().BoolVarP(&pruneQuiet, "quiet", "q", false, "Oculta a saída no terminal (requer -f ou -l)")
	rootCmd.AddCommand(pruneCmd)
}
