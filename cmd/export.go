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
	"tae/internal/render"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	exportZip     bool
	exportLimit   int
	exportMerge   bool
	exportFlatten bool
	exportQuiet   bool
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

		rawFiles, err := storage.GetFilesByTag(tagName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao buscar rastreamento da tag: %v\n", err)
			os.Exit(1)
		}
		if len(rawFiles) == 0 {
			fmt.Printf("A tag '%s' não possui alvos rastreados ou não existe.\n", tagName)
			os.Exit(1)
		}

		resolvedFiles, err := restorePathsForDisk(tagName, rawFiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro de escopo estrutural: %v\n", err)
			os.Exit(1)
		}

		ignoredMap, err := storage.GetIgnoredPaths(tagName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Aviso: Falha ao carregar Exclusion Index: %v\n", err)
		}

		restoredIgnored := make(map[string]bool)
		var igPaths []string
		for p := range ignoredMap {
			igPaths = append(igPaths, p)
		}
		if resIgPaths, err := restorePathsForDisk(tagName, igPaths); err == nil {
			for _, p := range resIgPaths {
				restoredIgnored[p] = true
			}
		}

		files := expandPathsToFiles(resolvedFiles, restoredIgnored)
		if len(files) == 0 {
			fmt.Println("Erro: Nenhum arquivo válido encontrado (possivelmente todos foram ignorados).")
			os.Exit(1)
		}

		if err := os.MkdirAll(destPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar destino: %v\n", err)
			os.Exit(1)
		}

		basePrefix := render.GetCommonPrefix(files)
		numWorkers := runtime.NumCPU()

		var flattenMap map[string]string
		if exportFlatten {
			flattenMap = render.ResolveFlattenNames(files, basePrefix)
		}

		var printMu sync.Mutex

		if exportZip {
			chunks := grouper.GroupFiles(files, exportLimit, tagName, exportMerge)
			fmt.Printf("Iniciando exportação em ZIP. %d arquivo(s) expandido(s) divididos em %d lote(s) para '%s'...\n", len(files), len(chunks), destPath)
			if !exportQuiet {
				fmt.Printf("[Raiz Comum: %s]\n\n", basePrefix)
			}

			jobs := make(chan grouper.ExportChunk, len(chunks))
			var wg sync.WaitGroup

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go zipWorker(jobs, &wg, basePrefix, destPath, flattenMap, &printMu)
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
				go flatWorker(jobs, &wg, basePrefix, destPath, flattenMap, &printMu)
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
	exportCmd.Flags().BoolVarP(&exportFlatten, "flatten", "f", false, "Exporta todos os arquivos no mesmo nível (sem pastas), resolvendo colisões de nomes")
	exportCmd.Flags().BoolVarP(&exportQuiet, "quiet", "q", false, "Oculta a listagem individual dos arquivos no console")
	rootCmd.AddCommand(exportCmd)
}

func expandPathsToFiles(paths []string, ignored map[string]bool) []string {
	uniqueFiles := make(map[string]bool)
	var expanded []string

	for _, p := range paths {
		if ignored[p] {
			continue
		}

		info, err := os.Stat(p)
		if err != nil {
			continue
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

			if ignored[path] {
				if f.IsDir() {
					return filepath.SkipDir
				}
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

func zipWorker(jobs <-chan grouper.ExportChunk, wg *sync.WaitGroup, basePrefix, dest string, flattenMap map[string]string, mu *sync.Mutex) {
	defer wg.Done()
	for chunk := range jobs {
		zipPath := filepath.Join(dest, chunk.ZipName)
		err := createZipChunk(zipPath, chunk.Files, basePrefix, flattenMap)

		mu.Lock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar %s: %v\n", chunk.ZipName, err)
		} else {
			fmt.Printf("  -> %s gerado (%d arquivos reais)\n", chunk.ZipName, len(chunk.Files))
			if !exportQuiet {
				for _, path := range chunk.Files {
					var relPath string
					if flattenMap != nil && flattenMap[path] != "" {
						relPath = flattenMap[path]
					} else {
						relPath = strings.TrimPrefix(path, basePrefix)
						relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
						if relPath == "" {
							relPath = filepath.Base(path)
						}
						relPath = filepath.ToSlash(relPath)
					}
					fmt.Printf("      - %s\n", relPath)
				}
			}
		}
		mu.Unlock()
	}
}

func createZipChunk(zipPath string, files []string, basePrefix string, flattenMap map[string]string) error {
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

		var relPath string
		if flattenMap != nil && flattenMap[path] != "" {
			relPath = flattenMap[path]
		} else {
			relPath = strings.TrimPrefix(path, basePrefix)
			relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
			if relPath == "" {
				relPath = filepath.Base(path)
			}
			relPath = filepath.ToSlash(relPath)
		}

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

func flatWorker(jobs <-chan string, wg *sync.WaitGroup, basePrefix, dest string, flattenMap map[string]string, mu *sync.Mutex) {
	defer wg.Done()
	for path := range jobs {
		var relPath string
		if flattenMap != nil && flattenMap[path] != "" {
			relPath = flattenMap[path]
		} else {
			relPath = strings.TrimPrefix(path, basePrefix)
			relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
			if relPath == "" {
				relPath = filepath.Base(path)
			}
		}

		targetPath := filepath.Join(dest, relPath)
		err := copySingleFile(path, targetPath)

		mu.Lock()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao exportar %s: %v\n", path, err)
		} else if !exportQuiet {
			fmt.Printf("  -> %s\n", targetPath)
		}
		mu.Unlock()
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
