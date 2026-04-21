// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package exporter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tae/internal/config"
	"tae/internal/render"
	"tae/internal/vcs"
)

// ExportSingleFile consolida todos os arquivos monitorados em um único arquivo texto plano otimizado para LLMs.
// É executado de forma sequencial intencionalmente para garantir ordenação determinística e evitar consumo excessivo de RAM.
func ExportSingleFile(destPath string, files []string, opts ExportOptions) error {
	filter, err := config.LoadFilter()
	if err != nil {
		return fmt.Errorf("falha na camada de configuração: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("erro de I/O ao criar diretório base: %w", err)
	}

	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo unificado: %w", err)
	}
	defer outFile.Close()

	var br *vcs.BatchReader
	if opts.GitCommit != "" {
		gitRoot := vcs.GetRoot()
		var errGit error
		br, errGit = vcs.NewBatchReader(gitRoot)
		if errGit != nil {
			return fmt.Errorf("falha crítica no motor do Git: %w", errGit)
		}
		defer br.Close()
	}

	// 1. Escreve Cabeçalho e Metadados
	fmt.Fprintln(outFile, "================================================================")
	fmt.Fprintln(outFile, " TAE Export - Single File")
	if opts.GitCommit != "" {
		fmt.Fprintf(outFile, " Commit Original: %s\n", opts.GitCommit)
	}
	fmt.Fprintln(outFile, "================================================================")
	fmt.Fprintln(outFile, "\n# Estrutura de Diretórios")
	fmt.Fprintln(outFile, "```")

	rootNode := render.BuildVisualTree(files, opts.BasePrefix)
	render.PrintTree(outFile, rootNode, "", 0, 0, nil)

	fmt.Fprintln(outFile, "```\n")
	fmt.Fprintln(outFile, "# Arquivos do Escopo")

	// Ordenação Hierárquica em Memória
	sort.Slice(files, func(i, j int) bool {
		relI := resolveRelPath(files[i], opts.BasePrefix, opts.FlattenMap)
		relJ := resolveRelPath(files[j], opts.BasePrefix, opts.FlattenMap)

		dirI := filepath.Dir(relI)
		dirJ := filepath.Dir(relJ)

		if dirI == dirJ {
			return filepath.Base(relI) < filepath.Base(relJ)
		}
		return dirI < dirJ
	})

	reader := bufio.NewReader(os.Stdin)

	// 2. Despeja o conteúdo de cada arquivo sequencialmente
	for _, path := range files {
		relPath := resolveRelPath(path, opts.BasePrefix, opts.FlattenMap)
		if opts.AppendTxt {
			relPath += ".txt"
		}

		fmt.Fprintln(outFile, "\n================================================================")
		fmt.Fprintf(outFile, "File: %s\n", relPath)
		fmt.Fprintln(outFile, "================================================================")

		ext := strings.ToLower(filepath.Ext(path))
		skip := false

		if ext != "" {
			if filter.Blocked[ext] {
				skip = true
			} else if !filter.Allowed[ext] {
				if opts.Quiet {
					// Em modo quiet, omite por segurança mas não polui o JSON com regras inferidas
					skip = true
				} else {
					fmt.Printf("\n[?] A extensão '%s' do arquivo '%s' é desconhecida.\n", ext, relPath)
					fmt.Printf("Deseja incluir seu conteúdo e PERMITIR essa extensão no futuro? [s/N]: ")
					
					response, _ := reader.ReadString('\n')
					response = strings.TrimSpace(strings.ToLower(response))
					
					if response == "s" || response == "y" {
						if err := filter.LearnExtension(ext, false); err != nil {
							fmt.Printf("Aviso: Falha ao salvar regra de permissão: %v\n", err)
						}
						skip = false
					} else {
						if err := filter.LearnExtension(ext, true); err != nil {
							fmt.Printf("Aviso: Falha ao salvar regra de bloqueio: %v\n", err)
						}
						skip = true
					}
				}
			}
		}

		if skip {
			fmt.Fprintln(outFile, "[Conteúdo de arquivo omitido: extensão não-texto bloqueada/desconhecida]")
			if !opts.Quiet {
				fmt.Printf("  -> Omitido: %s\n", relPath)
			}
			continue
		}

		err := writeContent(path, opts.GitCommit, outFile, br)
		if err != nil {
			fmt.Fprintf(outFile, "[Erro de I/O ao ler conteúdo deste arquivo: %v]\n", err)
			if !opts.Quiet {
				fmt.Printf("Aviso: Falha ao ler '%s': %v\n", relPath, err)
			}
		} else {
			fmt.Fprintln(outFile, "")
		}

		if !opts.Quiet {
			fmt.Printf("  -> Anexado: %s\n", relPath)
		}
	}

	return nil
}
