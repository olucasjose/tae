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
	"time"

	"tae/internal/grouper"
    "tae/internal/render"

	"github.com/spf13/cobra"
)

var (
	diffLimit int
	diffMerge bool
)

var gitDiffCmd = &cobra.Command{
	Use:   "diff <commit1> <commit2>",
	Short: "Compacta arquivos alterados entre dois commits isolado da working tree",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commit1, commit2 := args[0], args[1]
		fmt.Printf("Comparando %s -> %s\n\n", commit1, commit2)

		files := getChangedFiles(commit1, commit2)
		if len(files) == 0 {
			fmt.Println("\nNenhum arquivo encontrado para compactar.")
			os.Exit(0)
		}

		timestamp := time.Now().Format("20060102_150405")
		repoName := getGitRepoName()
		baseName := fmt.Sprintf("%s-diff-%s", repoName, timestamp)
		basePrefix := render.GetCommonPrefix(files)

		chunks := grouper.GroupFiles(files, diffLimit, baseName, diffMerge)
		fmt.Printf("\nIniciando empacotamento de %d arquivo(s) em %d lote(s)...\n", len(files), len(chunks))

		numWorkers := runtime.NumCPU()
		jobs := make(chan grouper.ExportChunk, len(chunks))
		var wg sync.WaitGroup

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go diffWorker(jobs, &wg, basePrefix, commit2)
		}

		for _, c := range chunks { jobs <- c }
		close(jobs)
		wg.Wait()

		fmt.Printf("\nSucesso! %d arquivo(s) zip gerado(s) no diretório atual.\n", len(chunks))
	},
}

func init() {
	gitDiffCmd.Flags().IntVarP(&diffLimit, "limit", "l", 0, "Teto máximo de arquivos por zip")
	gitDiffCmd.Flags().BoolVarP(&diffMerge, "merge", "m", false, "Mescla zips subpopulados mantendo o limite (requer -l)")
	gitCmd.AddCommand(gitDiffCmd)
}

func getChangedFiles(c1, c2 string) []string {
	cmd := exec.Command("git", "diff", "--name-status", c1, c2)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro no git diff:\n%s\n", stderr.String())
		os.Exit(1)
	}

	var filesToZip []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }
		
		parts := strings.Split(line, "\t")
		if len(parts) < 2 { continue }

		statusChar := strings.ToUpper(parts[0])[0]
		var filePath string
		isRename := false

		if (statusChar == 'A' || statusChar == 'M') && len(parts) >= 2 {
			filePath = parts[1]
		} else if statusChar == 'R' && len(parts) >= 3 {
			filePath = parts[2]
			isRename = true
		} else {
			continue // Ignora remoções (D) ou status desconhecidos
		}

		filesToZip = append(filesToZip, filePath)
		if isRename {
			fmt.Printf("  R: %s (renomeado)\n", filePath)
		} else {
			fmt.Printf("  %c: %s\n", statusChar, filePath)
		}
	}
	return filesToZip
}

func diffWorker(jobs <-chan grouper.ExportChunk, wg *sync.WaitGroup, basePrefix, targetCommit string) {
	defer wg.Done()
	for chunk := range jobs {
		if err := buildZipChunk(chunk.ZipName, chunk.Files, basePrefix, targetCommit); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar %s: %v\n", chunk.ZipName, err)
		} else {
			fmt.Printf("  -> %s gerado (%d arquivos)\n", chunk.ZipName, len(chunk.Files))
		}
	}
}

func buildZipChunk(zipPath string, files []string, basePrefix, commit string) error {
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
