// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package vcs

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DiffStatus representa o estado de um arquivo alterado no Git isolando a camada CLI de prints
type DiffStatus struct {
	Status   byte
	Path     string
	IsRename bool
}

// BatchReader gerencia um subprocesso persistente do git cat-file --batch
type BatchReader struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
}

// NewBatchReader inicializa o processo atado ao ciclo de vida do worker
func NewBatchReader(gitRoot string) (*BatchReader, error) {
	cmd := exec.Command("git", "cat-file", "--batch")
	cmd.Dir = gitRoot // Garante que roda no contexto do repositório correto

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("falha ao iniciar git cat-file --batch: %w", err)
	}

	return &BatchReader{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdoutPipe),
	}, nil
}

// ReadBlob extrai um arquivo específico do git usando o canal aberto
func (b *BatchReader) ReadBlob(commit, path string, dest io.Writer) error {
	gitPath := filepath.ToSlash(path)
	request := fmt.Sprintf("%s:%s\n", commit, gitPath)

	if _, err := b.stdin.Write([]byte(request)); err != nil {
		return fmt.Errorf("falha ao enviar requisição ao batch: %w", err)
	}

	// O formato de resposta padrão do git é: <oid> <type> <size>\n
	// Se o objeto não existir, retorna: <object> missing\n
	header, err := b.stdout.ReadString('\n')
	if err != nil {
		return fmt.Errorf("falha ao ler cabeçalho do batch: %w", err)
	}

	header = strings.TrimSpace(header)
	if strings.HasSuffix(header, "missing") {
		return fmt.Errorf("objeto não encontrado no git: %s", request)
	}

	parts := strings.Split(header, " ")
	if len(parts) < 3 {
		return fmt.Errorf("cabeçalho de resposta inválido: %s", header)
	}

	size, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return fmt.Errorf("tamanho de arquivo inválido no cabeçalho: %w", err)
	}

	// Copia exatamente 'size' bytes da stream para o destino
	if _, err := io.CopyN(dest, b.stdout, size); err != nil {
		return fmt.Errorf("falha ao ler corpo do blob: %w", err)
	}

	// Consome a quebra de linha residual após a leitura do conteúdo
	if _, err := b.stdout.ReadByte(); err != nil {
		return fmt.Errorf("falha ao ler byte residual: %w", err)
	}

	return nil
}

// Close encerra o processo de forma limpa, fechando o stdin
func (b *BatchReader) Close() error {
	b.stdin.Close()
	return b.cmd.Wait()
}

// IsInsideRepo verifica silenciosamente se o diretório atual é uma working tree válida
func IsInsideRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
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
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("falha ao criar pipe para git diff: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("erro ao iniciar git diff:\n%s", stderr.String())
	}

	var changes []DiffStatus
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erro na leitura da stream do git diff: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("erro no git diff:\n%s", stderr.String())
	}

	return changes, nil
}

// ListTree lista a árvore de arquivos de um commit
func ListTree(commit string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", commit)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("falha ao criar pipe para git ls-tree: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("erro ao iniciar git ls-tree:\n%s", stderr.String())
	}

	var files []string
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		f := strings.TrimSpace(scanner.Text())
		if f != "" {
			files = append(files, f)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("erro na leitura da stream do git ls-tree: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("erro ao ler árvore do Git. Verifique o repositório e o hash:\n%s", stderr.String())
	}

	return files, nil
}
