// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"tae/internal/grouper"
	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var (
	exportZip   bool
	exportLimit int
	exportMerge bool
)

var exportCmd = &cobra.Command{
	Use:   "export <nome da tag> <destino>",
	Short: "Exporta os arquivos e diretórios monitorados para um destino",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			tags, _ := storage.GetAllTags()
			return tags, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 {
			return nil, cobra.ShellCompDirectiveFilterDirs
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[0]
		destPath := args[1]

		rawFiles := getTagFiles(tagName)
		if len(rawFiles) == 0 {
			fmt.Printf("A tag '%s' não possui alvos rastreados ou não existe.\n", tagName)
			os.Exit(1)
		}

		// Pré-processamento: expande pastas em arquivos reais e deduplica
		files := expandPathsToFiles(rawFiles)
		if len(files) == 0 {
			fmt.Println("Erro: Nenhum arquivo válido encontrado nos alvos rastreados.")
			os.Exit(1)
		}

		if err := os.MkdirAll(destPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar destino: %v\n", err)
			os.Exit(1)
		}

		basePrefix := getCommonPrefix(files)
		numWorkers := runtime.NumCPU()

		if exportZip {
			chunks := grouper.GroupFiles(files, exportLimit, tagName, exportMerge)
			fmt.Printf("Iniciando exportação em ZIP. %d arquivo(s) expandido(s) divididos em %d lote(s) para '%s'...\n", len(files), len(chunks), destPath)

			jobs := make(chan grouper.ExportChunk, len(chunks))
			var wg sync.WaitGroup

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go zipWorker(jobs, &wg, basePrefix, destPath)
			}

			for _, c := range chunks {
				jobs <- c
			}
			close(jobs)
			wg.Wait()

			fmt.Printf("\nSucesso! %d arquivo(s) zip gerado(s) na pasta '%s'.\n", len(chunks), destPath)
		} else {
			fmt.Printf("Iniciando exportação plana de %d arquivo(s) para '%s' com %d worker(s)....\n", len(files), destPath, numWorkers)

			jobs := make(chan string, len(files))
			var wg sync.WaitGroup

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go flatWorker(jobs, &wg, basePrefix, destPath)
			}

			for _, file := range files {
				jobs <- file
			}
			close(jobs)
			wg.Wait()

			fmt.Printf("\nSucesso! Arquivos exportados para a pasta '%s'.\n", destPath)
		}
	},
}

func init() {
	exportCmd.Flags().BoolVarP(&exportZip, "zip", "z", false, "Exporta e compacta os arquivos em formato .zip")
	exportCmd.Flags().IntVarP(&exportLimit, "limit", "l", 0, "Teto máximo de arquivos por zip (requer -z)")
	exportCmd.Flags().BoolVarP(&exportMerge, "merge", "m", false, "Mescla zips subpopulados sequencialmente mantendo o limite (requer -z e -l)")
	rootCmd.AddCommand(exportCmd)
}

func getTagFiles(tagName string) []string {
	var files []string
	db, err := storage.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao abrir banco: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	db.View(func(tx *bbolt.Tx) error {
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
	return files
}

// expandPathsToFiles converte diretórios rastreados em uma lista plana de arquivos absolutos, evitando duplicações.
func expandPathsToFiles(paths []string) []string {
	uniqueFiles := make(map[string]bool)
	var expanded []string

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue // Ignora alvos que foram deletados do disco
		}

		if !info.IsDir() {
			if !uniqueFiles[p] {
				uniqueFiles[p] = true
				expanded = append(expanded, p)
			}
			continue
		}

		filepath.Walk(p, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !f.IsDir() {
				if !uniqueFiles[path] {
					uniqueFiles[path] = true
					expanded = append(expanded, path)
				}
			}
			return nil
		})
	}
	return expanded
}

func zipWorker(jobs <-chan grouper.ExportChunk, wg *sync.WaitGroup, basePrefix, dest string) {
	defer wg.Done()
	for chunk := range jobs {
		zipPath := filepath.Join(dest, chunk.ZipName)
		if err := createZipChunk(zipPath, chunk.Files, basePrefix); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar %s: %v\n", chunk.ZipName, err)
		} else {
			fmt.Printf("  -> %s gerado (%d arquivos reais)\n", chunk.ZipName, len(chunk.Files))
		}
	}
}

func createZipChunk(zipPath string, files []string, basePrefix string) error {
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

func flatWorker(jobs <-chan string, wg *sync.WaitGroup, basePrefix, dest string) {
	defer wg.Done()
	for path := range jobs {
		relPath := strings.TrimPrefix(path, basePrefix)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

		if relPath == "" {
			relPath = filepath.Base(path)
		}

		targetPath := filepath.Join(dest, relPath)

		if err := copySingleFile(path, targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao exportar %s: %v\n", path, err)
		} else {
			fmt.Printf("  -> %s\n", targetPath)
		}
	}
}

func copySingleFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func getCommonPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return filepath.Dir(paths[0]) + string(filepath.Separator)
	}

	prefix := strings.Split(paths[0], string(filepath.Separator))

	for _, p := range paths[1:] {
		parts := strings.Split(p, string(filepath.Separator))
		limit := len(prefix)
		if len(parts) < limit {
			limit = len(parts)
		}

		var i int
		for i = 0; i < limit; i++ {
			if prefix[i] != parts[i] {
				break
			}
		}
		prefix = prefix[:i]
	}

	return strings.Join(prefix, string(filepath.Separator)) + string(filepath.Separator)
}
