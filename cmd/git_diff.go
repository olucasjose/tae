// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"runtime"
	"time"

	"tae/internal/grouper"
	"tae/internal/render"
	"tae/internal/storage"
	"tae/internal/exporter"
	"tae/internal/vcs"

	"github.com/spf13/cobra"
)

var (
	diffLimit    int
	diffMerge    bool
	diffNoIgnore bool
)

var gitDiffCmd = &cobra.Command{
	Use:   "diff <commit1> <commit2>",
	Short: "Compacta arquivos alterados entre dois commits isolado da working tree",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		commit1, commit2 := args[0], args[1]
		fmt.Printf("Comparando %s -> %s\n\n", commit1, commit2)

		changes, err := vcs.GetChangedFiles(commit1, commit2)
		if err != nil {
			return err
		}

		if len(changes) == 0 {
			fmt.Println("\nNenhum arquivo modificado encontrado na comparação.")
			return nil
		}

		var rawFiles []string
		for _, c := range changes {
			rawFiles = append(rawFiles, c.Path)
			if c.IsRename {
				fmt.Printf("  R: %s (renomeado)\n", c.Path)
			} else {
				fmt.Printf("  %c: %s\n", c.Status, c.Path)
			}
		}

		var files []string
		if !diffNoIgnore {
			repoID := vcs.GetRepoID()
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				fmt.Printf("Aviso: Falha ao carregar denylist do repositório: %v\n", err)
			}

			for _, f := range rawFiles {
				if !isGitPathIgnored(f, ignoredMap) {
					files = append(files, f)
				} else {
					fmt.Printf("  I: %s (ignorado via denylist)\n", f)
				}
			}
		} else {
			files = rawFiles
		}

		if len(files) == 0 {
			fmt.Println("\nTodos os arquivos alterados foram retidos pela denylist. Nada a compactar.")
			return nil
		}

		timestamp := time.Now().Format("20060102_150405")
		repoName := vcs.GetRepoName()
		baseName := fmt.Sprintf("%s-diff-%s", repoName, timestamp)
		basePrefix := render.GetCommonPrefix(files)

		chunks := grouper.GroupFiles(files, diffLimit, baseName, diffMerge)
		fmt.Printf("\nIniciando empacotamento de %d arquivo(s) em %d lote(s)...\n", len(files), len(chunks))

		numWorkers := runtime.NumCPU()
		
		opts := exporter.ExportOptions{
			DestDir:    ".", // diff sempre exporta no diretório atual
			BasePrefix: basePrefix,
			GitCommit:  commit2,
		}
		exporter.ExportZip(chunks, numWorkers, opts)
		
		fmt.Printf("\nSucesso! %d arquivo(s) zip gerado(s) no diretório atual.\n", len(chunks))
		return nil
	},
}

func init() {
	gitDiffCmd.Flags().IntVarP(&diffLimit, "limit", "l", 0, "Teto máximo de arquivos por zip")
	gitDiffCmd.Flags().BoolVarP(&diffMerge, "merge", "m", false, "Mescla zips subpopulados mantendo o limite (requer -l)")
	gitDiffCmd.Flags().BoolVar(&diffNoIgnore, "no-ignore", false, "Ignora a denylist do repositório e empacota todos os arquivos alterados")
	gitCmd.AddCommand(gitDiffCmd)
}
