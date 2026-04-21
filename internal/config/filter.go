// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FilterConfig mapeia a estrutura exata do JSON no disco
type FilterConfig struct {
	BlockedExtensions []string `json:"blocked_extensions"`
	AllowedExtensions []string `json:"allowed_extensions"`
}

// ExtensionFilter encapsula o estado em memória e os métodos de I/O
type ExtensionFilter struct {
	configPath string
	config     *FilterConfig
	Blocked    map[string]bool
	Allowed    map[string]bool
}

var defaultBlocked = []string{
	".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".ico", ".webp",
	".mp4", ".avi", ".mkv", ".mov", ".wmv", ".webm",
	".mp3", ".wav", ".flac", ".aac", ".ogg",
	".pdf", ".zip", ".tar", ".gz", ".rar", ".7z",
	".exe", ".dll", ".so", ".dylib", ".bin",
	".ttf", ".otf", ".woff", ".woff2", ".eot",
}

var defaultAllowed = []string{
	".go", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h", ".hpp", ".cs",
	".php", ".rb", ".rs", ".swift", ".kt", ".kts", ".scala", ".sh", ".bash",
	".html", ".css", ".scss", ".md", ".txt", ".json", ".xml", ".yaml", ".yml",
	".ini", ".toml", ".csv", ".sql", ".mod", ".sum", ".env", ".gitignore",
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("falha ao localizar diretório home: %w", err)
	}

	dir := filepath.Join(home, ".tae")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório base de configuração: %w", err)
	}

	return filepath.Join(dir, "single_file_filter.json"), nil
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

// LoadFilter inicializa a configuração, criando o arquivo padrão se não existir, e constrói os índices O(1).
func LoadFilter() (*ExtensionFilter, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &FilterConfig{}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg.BlockedExtensions = defaultBlocked
		cfg.AllowedExtensions = defaultAllowed
		
		sort.Strings(cfg.BlockedExtensions)
		sort.Strings(cfg.AllowedExtensions)

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("erro ao gerar json padrão de extensões: %w", err)
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return nil, fmt.Errorf("erro ao salvar filtro de extensões no disco: %w", err)
		}
	} else {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler filtro de extensões: %w", err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("JSON inválido no arquivo de filtro ~/.tae/single_file_filter.json: %w", err)
		}
	}

	filter := &ExtensionFilter{
		configPath: configPath,
		config:     cfg,
		Blocked:    make(map[string]bool),
		Allowed:    make(map[string]bool),
	}

	for _, ext := range cfg.BlockedExtensions {
		if norm := normalizeExt(ext); norm != "" {
			filter.Blocked[norm] = true
		}
	}
	for _, ext := range cfg.AllowedExtensions {
		if norm := normalizeExt(ext); norm != "" {
			filter.Allowed[norm] = true
		}
	}

	return filter, nil
}

// LearnExtension atualiza as fatias base, os índices O(1) e reescreve o JSON no disco instantaneamente.
func (f *ExtensionFilter) LearnExtension(ext string, block bool) error {
	norm := normalizeExt(ext)
	if norm == "" {
		return nil // Ignora aprendizado de arquivos sem extensão
	}

	// Previne duplicações lógicas
	if f.Blocked[norm] || f.Allowed[norm] {
		return nil
	}

	if block {
		f.config.BlockedExtensions = append(f.config.BlockedExtensions, norm)
		f.Blocked[norm] = true
	} else {
		f.config.AllowedExtensions = append(f.config.AllowedExtensions, norm)
		f.Allowed[norm] = true
	}

	sort.Strings(f.config.BlockedExtensions)
	sort.Strings(f.config.AllowedExtensions)

	data, err := json.MarshalIndent(f.config, "", "  ")
	if err != nil {
		return fmt.Errorf("erro de serialização ao aprender extensão '%s': %w", norm, err)
	}

	if err := os.WriteFile(f.configPath, data, 0644); err != nil {
		return fmt.Errorf("erro de I/O ao salvar aprendizado da extensão '%s': %w", norm, err)
	}

	return nil
}
