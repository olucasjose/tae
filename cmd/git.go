// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Agrupa comandos relacionados a operações do repositório Git",
	Long:  "Comandos utilitários para integração com o Git, permitindo listar, exportar e gerar diffs empacotados.",
}

func init() {
	rootCmd.AddCommand(gitCmd)
}

// streamGitBlob lê os bytes diretamente dos objetos internos do Git (isolado do disco rígido)
// e os jorra no io.Writer de destino (que pode ser um buffer de Zip ou um arquivo local vazio)
func streamGitBlob(commit, path string, dest io.Writer) error {
	gitPath := filepath.ToSlash(path) // Garante o padrão UNIX exigido pelo Git
	
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commit, gitPath))
	cmd.Stdout = dest // Streaming direto, zero desperdício de memória RAM
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha ao ler blob do git: %s (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
