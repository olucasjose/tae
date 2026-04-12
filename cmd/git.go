// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Agrupa comandos relacionados a operações do repositório Git",
	Long:  "Comandos utilitários para integração com o Git, permitindo listar, exportar e gerar diffs empacotados.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !isInsideGitRepo() {
			fmt.Fprintln(os.Stderr, "⚠️ Alerta: O diretório atual não pertence a um repositório Git.")
			fmt.Fprintln(os.Stderr, "Navegue até a raiz ou subdiretório de um repositório válido antes de usar os comandos 'tae git'.")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)
}

// isInsideGitRepo verifica silenciosamente se o diretório atual é uma working tree válida.
func isInsideGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
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

// getGitRepoName extrai o nome do diretório raiz do repositório Git atual
func getGitRepoName() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "repo" // Fallback seguro
	}
	return filepath.Base(strings.TrimSpace(string(out)))
}

// getGitRepoID extrai o hash do commit raiz (imutável). Ele é nossa chave primária no banco.
func getGitRepoID() string {
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		// Fallback para o nome da pasta caso o repo seja recém-criado (sem commits)
		return getGitRepoName()
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		return lines[0]
	}
	return getGitRepoName()
}

// getGitRelativePath normaliza qualquer alvo do usuário (caminhos absolutos ou relativos)
// transformando-os em caminhos relativos à raiz do repositório, com barras '/'
func getGitRelativePath(target string) (string, error) {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("falha ao localizar raiz do git: %v", err)
	}
	gitRoot := strings.TrimSpace(string(out))
	
	if !strings.HasPrefix(absTarget, gitRoot) {
		return "", fmt.Errorf("o alvo '%s' encontra-se fora do repositório atual", target)
	}
	
	relPath := strings.TrimPrefix(absTarget, gitRoot)
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	
	return filepath.ToSlash(relPath), nil
}

// isGitPathIgnored verifica se o alvo exato ou algum de seus diretórios pais está na denylist.
func isGitPathIgnored(target string, ignoredMap map[string]bool) bool {
	if ignoredMap[target] {
		return true
	}
	parts := strings.Split(target, "/")
	current := ""
	for i := 0; i < len(parts)-1; i++ {
		if current == "" {
			current = parts[i]
		} else {
			current = current + "/" + parts[i]
		}
		if ignoredMap[current] {
			return true
		}
	}
	return false
}
