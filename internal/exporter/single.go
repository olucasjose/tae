// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package exporter

import (
	"fmt"
	"os"
	"path/filepath"

	"tae/internal/render"
	"tae/internal/vcs"
)

// ExportSingleFile consolida todos os arquivos monitorados em um único arquivo texto plano otimizado para LLMs.
// É executado de forma sequencial intencionalmente para garantir ordenação determinística e evitar consumo excessivo de RAM.
func ExportSingleFile(destPath string, files []string, opts ExportOptions) error {
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

	// 2. Despeja o conteúdo de cada arquivo sequencialmente
	for _, path := range files {
		relPath := resolveRelPath(path, opts.BasePrefix, opts.FlattenMap)
		if opts.AppendTxt {
			relPath += ".txt"
		}

		fmt.Fprintln(outFile, "\n================================================================")
		fmt.Fprintf(outFile, "File: %s\n", relPath)
		fmt.Fprintln(outFile, "================================================================")

		err := writeContent(path, opts.GitCommit, outFile, br)
		if err != nil {
			// Não abortamos o pipeline por 1 arquivo quebrado. Logamos o erro in-file para o LLM ter contexto.
			fmt.Fprintf(outFile, "[Erro de I/O ao ler conteúdo deste arquivo: %v]\n", err)
			if !opts.Quiet {
				fmt.Printf("Aviso: Falha ao ler '%s': %v\n", relPath, err)
			}
		} else {
			fmt.Fprintln(outFile, "") // Pula uma linha no final para não emendar com a próxima assinatura
		}

		if !opts.Quiet {
			fmt.Printf("  -> Anexado: %s\n", relPath)
		}
	}

	return nil
}
