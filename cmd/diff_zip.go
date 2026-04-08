// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"tae/internal/grouper"

	"github.com/spf13/cobra"
)

var (
	diffLimit int
	diffMerge bool
)

var diffZipCmd = &cobra.Command{
	Use:   "diff-zip <commit1> <commit2>",
	Short: "Compacta arquivos alterados entre dois commits do Git no disco",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commit1 := args[0]
		commit2 := args[1]

		fmt.Printf("Comparando %s -> %s\n\n", commit1, commit2)

		files := getChangedFiles(commit1, commit2)
		if len(files) == 0 {
			fmt.Println("\nNenhum arquivo encontrado para compactar no disco.")
			os.Exit(0)
		}

		timestamp := time.Now().Format("20060102_150405")
		baseName := fmt.Sprintf("git_changes_%s", timestamp)
		basePrefix := getCommonPrefix(files)

		chunks := grouper.GroupFiles(files, diffLimit, baseName, diffMerge)
		fmt.Printf("\nIniciando empacotamento de %d arquivo(s) em %d lote(s)...\n", len(files), len(chunks))

		numWorkers := runtime.NumCPU()
		jobs := make(chan grouper.ExportChunk, len(chunks))
		var wg sync.WaitGroup

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go diffWorker(jobs, &wg, basePrefix)
		}

		for _, c := range chunks {
			jobs <- c
		}
		close(jobs)
		wg.Wait()

		fmt.Printf("\nSucesso! %d arquivo(s) zip gerado(s) no diretório atual.\n", len(chunks))
	},
}

func init() {
	diffZipCmd.Flags().IntVarP(&diffLimit, "limit", "l", 0, "Teto máximo de arquivos por zip")
	diffZipCmd.Flags().BoolVarP(&diffMerge, "merge", "m", false, "Mescla zips subpopulados sequencialmente mantendo o limite (requer -l)")
	rootCmd.AddCommand(diffZipCmd)
}

func getChangedFiles(c1, c2 string) []string {
	cmd := exec.Command("git", "diff", "--name-status", c1, c2)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao executar git diff. Certifique-se de que está em um repositório Git válido.\n%s\n", stderr.String())
		os.Exit(1)
	}

	var filesToZip []string
	lines := strings.Split(out.String(), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		statusCode := strings.ToUpper(parts[0])
		statusChar := statusCode[0]

		var filePath string
		isRename := false

		if (statusChar == 'A' || statusChar == 'M') && len(parts) >= 2 {
			filePath = parts[1]
		} else if statusChar == 'R' && len(parts) >= 3 {
			filePath = parts[2]
			isRename = true
		} else {
			continue
		}

		info, err := os.Stat(filePath)
		if err == nil && !info.IsDir() {
			filesToZip = append(filesToZip, filePath)
			if isRename {
				fmt.Printf("  R: %s (renomeado)\n", filePath)
			} else {
				fmt.Printf("  %c: %s\n", statusChar, filePath)
			}
		}
	}

	return filesToZip
}

func diffWorker(jobs <-chan grouper.ExportChunk, wg *sync.WaitGroup, basePrefix string) {
	defer wg.Done()
	for chunk := range jobs {
		if err := buildZipChunk(chunk.ZipName, chunk.Files, basePrefix); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar %s: %v\n", chunk.ZipName, err)
		} else {
			fmt.Printf("  -> %s gerado (%d arquivos)\n", chunk.ZipName, len(chunk.Files))
		}
	}
}

func buildZipChunk(zipPath string, files []string, basePrefix string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}

		relPath := strings.TrimPrefix(path, basePrefix)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		if relPath == "" {
			relPath = filepath.Base(path)
		}
		relPath = filepath.ToSlash(relPath)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		fileToZip, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(writer, fileToZip)
		fileToZip.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
