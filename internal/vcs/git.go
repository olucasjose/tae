// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package vcs

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiffStatus representa o estado de um arquivo alterado no Git isolando a camada CLI de prints
type DiffStatus struct {
	Status   byte
	Path     string
	IsRename bool
}

// IsInsideRepo verifica silenciosamente se o diretório atual é uma working tree válida
func IsInsideRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

// StreamBlob lê os bytes diretamente dos objetos internos do Git (isolado do disco rígido)
func StreamBlob(commit, path string, dest io.Writer) error {
	gitPath := filepath.ToSlash(path) // Garante o padrão UNIX exigido pelo Git
	
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commit, gitPath))
	cmd.Stdout = dest // Streaming direto
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha ao ler blob do git: %s (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// GetRepoName extrai o nome do diretório raiz do repositório atual
func GetRepoName() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "repo"
	}
	return filepath.Base(strings.TrimSpace(string(out)))
}

// GetRepoID extrai o hash do commit raiz (imutável)
func GetRepoID() string {
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return GetRepoName()
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) > 0 && lines[0] != "" {
		return lines[0]
	}
	return GetRepoName()
}

// GetRoot retorna o caminho absoluto da raiz do repositório
func GetRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return filepath.ToSlash(strings.TrimSpace(string(out)))
}

// GetRelativePath normaliza qualquer alvo do usuário garantindo escopo no repositório
func GetRelativePath(target string) (string, error) {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	
	gitRoot := GetRoot()
	if gitRoot == "" {
		return "", fmt.Errorf("falha ao localizar raiz do git")
	}
	
	if !strings.HasPrefix(absTarget, gitRoot) {
		return "", fmt.Errorf("o alvo '%s' encontra-se fora do repositório atual", target)
	}
	
	relPath := strings.TrimPrefix(absTarget, gitRoot)
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	
	return filepath.ToSlash(relPath), nil
}

// GetChangedFiles retorna a lista e o status de arquivos alterados entre dois commits
func GetChangedFiles(c1, c2 string) ([]DiffStatus, error) {
	cmd := exec.Command("git", "diff", "--name-status", c1, c2)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("erro no git diff:\n%s", stderr.String())
	}

	var changes []DiffStatus
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		statusChar := strings.ToUpper(parts[0])[0]
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

		changes = append(changes, DiffStatus{
			Status:   statusChar,
			Path:     filePath,
			IsRename: isRename,
		})
	}
	return changes, nil
}

// ListTree lista a árvore de arquivos de um commit
func ListTree(commit string) ([]string, error) {
	gitExec := exec.Command("git", "ls-tree", "-r", "--name-only", commit)
	var out bytes.Buffer
	gitExec.Stdout = &out
	if err := gitExec.Run(); err != nil {
		return nil, fmt.Errorf("erro ao ler árvore do Git. Verifique o repositório e o hash")
	}

	var files []string
	for _, f := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}
