// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package exporter

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tae/internal/grouper"
	"tae/internal/vcs"
)

// ExportOptions define o comportamento declarativo do pipeline de exportação
type ExportOptions struct {
	DestDir    string
	BasePrefix string
	FlattenMap map[string]string
	Quiet      bool
	GitCommit  string // Se preenchido, o pipeline lê do histórico do Git em vez do disco rígido
}

func resolveRelPath(path, basePrefix string, flattenMap map[string]string) string {
	if flattenMap != nil && flattenMap[path] != "" {
		return flattenMap[path]
	}
	relPath := filepath.ToSlash(strings.TrimPrefix(path, basePrefix))
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" {
		relPath = filepath.Base(path)
	}
	return relPath
}

// writeContent recebe a instância ativa do BatchReader do worker atual
func writeContent(path, gitCommit string, w io.Writer, br *vcs.BatchReader) error {
	if gitCommit != "" && br != nil {
		return br.ReadBlob(gitCommit, path, w)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// ExportZip orquestra múltiplos workers para gerar lotes ZIP de forma concorrente
func ExportZip(chunks []grouper.ExportChunk, workers int, opts ExportOptions) {
	jobs := make(chan grouper.ExportChunk, len(chunks))
	logChan := make(chan string, 100)
	var wg sync.WaitGroup
	var logWg sync.WaitGroup

	// Goroutine dedicada para I/O no terminal (Fan-In)
	logWg.Add(1)
	go func() {
		defer logWg.Done()
		for msg := range logChan {
			fmt.Print(msg)
		}
	}()

	var gitRoot string
	if opts.GitCommit != "" {
		gitRoot = vcs.GetRoot()
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var br *vcs.BatchReader
			if opts.GitCommit != "" {
				var err error
				br, err = vcs.NewBatchReader(gitRoot)
				if err != nil {
					logChan <- fmt.Sprintf("Falha crítica ao iniciar worker do Git: %v\n", err)
					return
				}
				defer br.Close()
			}

			for chunk := range jobs {
				zipPath := filepath.Join(opts.DestDir, chunk.ZipName)
				err := buildZip(zipPath, chunk.Files, opts, br)

				var sb strings.Builder
				if err != nil {
					sb.WriteString(fmt.Sprintf("Erro ao criar %s: %v\n", chunk.ZipName, err))
				} else {
					sb.WriteString(fmt.Sprintf("  -> %s gerado (%d arquivos)\n", chunk.ZipName, len(chunk.Files)))
					if !opts.Quiet {
						for _, p := range chunk.Files {
							rel := resolveRelPath(p, opts.BasePrefix, opts.FlattenMap)
							sb.WriteString(fmt.Sprintf("      - %s\n", rel))
						}
					}
				}
				logChan <- sb.String()
			}
		}()
	}

	for _, c := range chunks {
		jobs <- c
	}
	close(jobs)
	wg.Wait()      // Aguarda todos os workers finalizarem a compressão
	close(logChan) // Sinaliza à goroutine de log que não haverá mais envios
	logWg.Wait()   // Aguarda o flush final no terminal
}

func buildZip(zipPath string, files []string, opts ExportOptions, br *vcs.BatchReader) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, path := range files {
		relPath := resolveRelPath(path, opts.BasePrefix, opts.FlattenMap)
		var writer io.Writer

		if opts.GitCommit == "" {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = relPath
			header.Method = zip.Deflate
			writer, err = archive.CreateHeader(header)
			if err != nil {
				return err
			}
		} else {
			writer, err = archive.Create(relPath)
			if err != nil {
				return err
			}
		}

		if err := writeContent(path, opts.GitCommit, writer, br); err != nil {
			return err
		}
	}
	return nil
}

// ExportFlat orquestra cópias planas de arquivos reconstruindo ou nivelando a árvore
func ExportFlat(files []string, workers int, opts ExportOptions) {
	jobs := make(chan string, len(files))
	logChan := make(chan string, 100)
	var wg sync.WaitGroup
	var logWg sync.WaitGroup

	// Goroutine dedicada para I/O no terminal (Fan-In)
	logWg.Add(1)
	go func() {
		defer logWg.Done()
		for msg := range logChan {
			fmt.Print(msg)
		}
	}()

	var gitRoot string
	if opts.GitCommit != "" {
		gitRoot = vcs.GetRoot()
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var br *vcs.BatchReader
			if opts.GitCommit != "" {
				var err error
				br, err = vcs.NewBatchReader(gitRoot)
				if err != nil {
					logChan <- fmt.Sprintf("Falha crítica ao iniciar worker do Git: %v\n", err)
					return
				}
				defer br.Close()
			}

			for path := range jobs {
				if opts.GitCommit == "" {
					info, err := os.Stat(path)
					if err != nil || info.IsDir() {
						continue
					}
				}

				relPath := resolveRelPath(path, opts.BasePrefix, opts.FlattenMap)
				targetPath := filepath.Join(opts.DestDir, relPath)

				var errOut error
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					errOut = fmt.Errorf("Erro de I/O em diretório %s: %v\n", path, err)
				} else {
					destFile, err := os.Create(targetPath)
					if err != nil {
						errOut = fmt.Errorf("Erro ao criar %s: %v\n", path, err)
					} else {
						errOut = writeContent(path, opts.GitCommit, destFile, br)
						destFile.Close()
					}
				}

				if errOut != nil {
					logChan <- fmt.Sprintf("%v\n", errOut)
				} else if !opts.Quiet {
					logChan <- fmt.Sprintf("  -> %s\n", targetPath)
				}
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)
	wg.Wait()      // Aguarda todos os workers concluírem a cópia
	close(logChan) // Sinaliza encerramento de logs
	logWg.Wait()   // Aguarda impressão das mensagens finais
}
