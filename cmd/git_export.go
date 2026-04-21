// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"tae/internal/vcs"
	"time"

	"tae/internal/exporter"
	"tae/internal/filter"
	"tae/internal/grouper"
	"tae/internal/render"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	gitExportZip      bool
	gitExportLimit    int
	gitExportMerge    bool
	gitExportNoIgnore bool
	gitExportFlatten  bool
	gitExportQuiet    bool
	gitExportTxt      bool
	gitExportSingle   bool
)

var gitExportCmd = &cobra.Command{
	Use:   "export <commit> <dest>",
	Short: "Exporta a árvore de arquivos de um commit isolado da working tree",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		commit := args[0]
		destPath := args[1]

		gitExec := exec.Command("git", "ls-tree", "-r", "--name-only", commit)
		var out bytes.Buffer
		gitExec.Stdout = &out
		if gitExportSingle && (gitExportZip || gitExportFlatten) {
			return fmt.Errorf("a flag --single-file (-s) é exclusiva e não pode ser usada simultaneamente com --zip ou --flatten")
		}
		if err := gitExec.Run(); err != nil {
			return fmt.Errorf("erro ao ler árvore do Git. Verifique o repositório e o hash")
		}

		var rawFiles []string
		for _, f := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			if f != "" {
				rawFiles = append(rawFiles, f)
			}
		}

		var files []string
		if !gitExportNoIgnore {
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
			return fmt.Errorf("nenhum arquivo válido encontrado para exportação (ou todos foram retidos pela denylist)")
		}

		if err := os.MkdirAll(destPath, 0755); err != nil {
			return fmt.Errorf("erro ao criar destino: %w", err)
		}

		basePrefix := render.GetCommonPrefix(files)
		numWorkers := runtime.NumCPU()

		var flattenMap map[string]string
		if gitExportFlatten {
			flattenMap = render.ResolveFlattenNames(files, basePrefix)
		}

		opts := exporter.ExportOptions{
			DestDir:    destPath,
			BasePrefix: basePrefix,
			FlattenMap: flattenMap,
			Quiet:      gitExportQuiet,
			GitCommit:  commit,
			AppendTxt:  gitExportTxt,
		}

		if gitExportSingle {
			timestamp := time.Now().Format("20060102_150405")
			fileName := fmt.Sprintf("%s_%s.txt", commit, timestamp)
			fullPath := filepath.Join(destPath, fileName)

			fmt.Printf("Iniciando exportação Single-File (Single File) do commit %s. %d arquivo(s) para '%s'...\n", commit, len(files), fullPath)
			if !gitExportQuiet {
				fmt.Printf("[Raiz Comum: %s]\n\n", basePrefix)
			}
			if err := exporter.ExportSingleFile(fullPath, files, opts); err != nil {
				return err
			}
			fmt.Printf("\nSucesso! Arquivo consolidado gerado em '%s'.\n", fullPath)
		} else if gitExportZip {
			repoName := vcs.GetRepoName()
			baseName := fmt.Sprintf("%s-%s", repoName, commit)
			chunks := grouper.GroupFiles(files, gitExportLimit, baseName, gitExportMerge)

			fmt.Printf("Iniciando exportação em ZIP do commit %s. %d arquivo(s) em %d lote(s)...\n", commit, len(files), len(chunks))
			if !gitExportQuiet {
				fmt.Printf("[Raiz Comum: %s]\n\n", basePrefix)
			}
			exporter.ExportZip(chunks, numWorkers, opts)
			fmt.Printf("\nSucesso! %d arquivo(s) zip gerado(s) em '%s'.\n", len(chunks), destPath)
		} else {
			fmt.Printf("Iniciando exportação plana de %d arquivo(s) para '%s'...\n", len(files), destPath)
			exporter.ExportFlat(files, numWorkers, opts)
			fmt.Printf("\nSucesso! Arquivos exportados para '%s'.\n", destPath)
		}
		return nil
	},
}

func init() {
	gitExportCmd.Flags().BoolVarP(&gitExportZip, "zip", "z", false, "Exporta e compacta os arquivos em formato .zip")
	gitExportCmd.Flags().IntVarP(&gitExportLimit, "limit", "l", 0, "Teto máximo de arquivos por zip (requer -z)")
	gitExportCmd.Flags().BoolVarP(&gitExportMerge, "merge", "m", false, "Mescla zips subpopulados mantendo o limite (requer -z e -l)")
	gitExportCmd.Flags().BoolVar(&gitExportNoIgnore, "no-ignore", false, "Ignora a denylist do repositório e exporta todos os arquivos")
	gitExportCmd.Flags().BoolVarP(&gitExportFlatten, "flatten", "f", false, "Exporta todos os arquivos no mesmo nível (sem pastas), resolvendo colisões de nomes")
	gitExportCmd.Flags().BoolVarP(&gitExportQuiet, "quiet", "q", false, "Oculta a listagem individual dos arquivos no console")
	gitExportCmd.Flags().BoolVar(&gitExportTxt, "txt", false, "Adiciona a extensão .txt a todos os arquivos exportados")
	gitExportCmd.Flags().BoolVarP(&gitExportSingle, "single-file", "s", false, "Exporta todos os arquivos em um único arquivo de texto plano (Single File)")
	gitCmd.AddCommand(gitExportCmd)
}
