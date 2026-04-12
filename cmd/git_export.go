// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

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
)

var gitExportCmd = &cobra.Command{
	Use:   "export <commit> <dest>",
	Short: "Exporta a árvore de arquivos de um commit isolado da working tree",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commit := args[0]
		destPath := args[1]

		gitExec := exec.Command("git", "ls-tree", "-r", "--name-only", commit)
		var out bytes.Buffer
		gitExec.Stdout = &out
		if err := gitExec.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Erro ao ler árvore do Git. Verifique o repositório e o hash.")
			os.Exit(1)
		}

		var rawFiles []string
		for _, f := range strings.Split(strings.TrimSpace(out.String()), "\n") {
			if f != "" {
				rawFiles = append(rawFiles, f)
			}
		}

		// Interceptação e Filtro da Denylist
		var files []string
		if !gitExportNoIgnore {
			repoID := getGitRepoID()
			ignoredMap, err := storage.GetGitIgnoredPaths(repoID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Aviso: Falha ao carregar denylist do repositório: %v\n", err)
			}
			
			for _, f := range rawFiles {
				if !isGitPathIgnored(f, ignoredMap) {
					files = append(files, f)
				}
			}
		} else {
			files = rawFiles
		}

		if len(files) == 0 {
			fmt.Println("Nenhum arquivo válido encontrado para exportação (ou todos foram retidos pela denylist).")
			os.Exit(1)
		}

		if err := os.MkdirAll(destPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar destino: %v\n", err)
			os.Exit(1)
		}

		basePrefix := render.GetCommonPrefix(files)
		numWorkers := runtime.NumCPU()

		if gitExportZip {
			repoName := getGitRepoName()
			baseName := fmt.Sprintf("%s-%s", repoName, commit)

			chunks := grouper.GroupFiles(files, gitExportLimit, baseName, gitExportMerge)
			fmt.Printf("Iniciando exportação em ZIP do commit %s. %d arquivo(s) em %d lote(s)...\n", commit, len(files), len(chunks))

			jobs := make(chan grouper.ExportChunk, len(chunks))
			var wg sync.WaitGroup

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go gitZipWorker(jobs, &wg, basePrefix, destPath, commit)
			}

			for _, c := range chunks { jobs <- c }
			close(jobs)
			wg.Wait()
			fmt.Printf("\nSucesso! %d arquivo(s) zip gerado(s) em '%s'.\n", len(chunks), destPath)
		} else {
			fmt.Printf("Iniciando exportação plana de %d arquivo(s) para '%s'...\n", len(files), destPath)

			jobs := make(chan string, len(files))
			var wg sync.WaitGroup

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go gitFlatWorker(jobs, &wg, basePrefix, destPath, commit)
			}

			for _, f := range files { jobs <- f }
			close(jobs)
			wg.Wait()
			fmt.Printf("\nSucesso! Arquivos exportados para '%s'.\n", destPath)
		}
	},
}

func init() {
	gitExportCmd.Flags().BoolVarP(&gitExportZip, "zip", "z", false, "Exporta e compacta os arquivos em formato .zip")
	gitExportCmd.Flags().IntVarP(&gitExportLimit, "limit", "l", 0, "Teto máximo de arquivos por zip (requer -z)")
	gitExportCmd.Flags().BoolVarP(&gitExportMerge, "merge", "m", false, "Mescla zips subpopulados mantendo o limite (requer -z e -l)")
	gitExportCmd.Flags().BoolVar(&gitExportNoIgnore, "no-ignore", false, "Ignora a denylist do repositório e exporta todos os arquivos")
	gitCmd.AddCommand(gitExportCmd)
}

func gitZipWorker(jobs <-chan grouper.ExportChunk, wg *sync.WaitGroup, basePrefix, dest, commit string) {
	defer wg.Done()
	for chunk := range jobs {
		zipPath := filepath.Join(dest, chunk.ZipName)
		if err := createGitZipChunk(zipPath, chunk.Files, basePrefix, commit); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar %s: %v\n", chunk.ZipName, err)
		} else {
			fmt.Printf("  -> %s gerado (%d arquivos)\n", chunk.ZipName, len(chunk.Files))
		}
	}
}

func createGitZipChunk(zipPath string, files []string, basePrefix, commit string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil { return err }
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, path := range files {
		relPath := filepath.ToSlash(strings.TrimPrefix(path, basePrefix))
		if relPath == "" { relPath = filepath.Base(path) }

		writer, err := archive.Create(relPath)
		if err != nil { return err }

		if err := streamGitBlob(commit, path, writer); err != nil {
			return err
		}
	}
	return nil
}

func gitFlatWorker(jobs <-chan string, wg *sync.WaitGroup, basePrefix, dest, commit string) {
	defer wg.Done()
	for path := range jobs {
		relPath := strings.TrimPrefix(path, basePrefix)
		targetPath := filepath.Join(dest, relPath)

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro de I/O em diretório %s: %v\n", path, err)
			continue
		}

		destFile, err := os.Create(targetPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar %s: %v\n", path, err)
			continue
		}

		if err := streamGitBlob(commit, path, destFile); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao exportar conteúdo de %s: %v\n", path, err)
		}
		destFile.Close()
	}
}
