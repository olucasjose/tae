// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"tae/internal/storage"
	"tae/internal/vcs"
)

// ResolveTagPaths consulta o metadado da tag e resolve os caminhos com base no escopo
func ResolveTagPaths(tagName string, targets []string) ([]string, error) {
	meta, err := storage.GetTagMeta(tagName)
	if err != nil {
		return nil, err
	}

	var resolved []string
	if meta.Type == storage.TagTypeGit {
		if !vcs.IsInsideRepo() {
			return nil, fmt.Errorf("a tag '%s' pertence ao Git, mas você não está em um repositório", tagName)
		}
		currentRepoID := vcs.GetRepoID()
		if meta.RepoID != "" && meta.RepoID != currentRepoID {
			displayName := meta.RepoName
			if displayName == "" {
				displayName = meta.RepoID
			} // Fallback
			return nil, fmt.Errorf("a tag '%s' pertence a outro repositório Git (%s)", tagName, displayName)
		}

		for _, t := range targets {
			relPath, err := vcs.GetRelativePath(t)
			if err != nil {
				return nil, fmt.Errorf("falha no alvo '%s': %w", t, err)
			}
			resolved = append(resolved, relPath)
		}
	} else {
		for _, t := range targets {
			absPath, err := filepath.Abs(t)
			if err != nil {
				return nil, fmt.Errorf("caminho inválido '%s': %w", t, err)
			}
			resolved = append(resolved, absPath)
		}
	}
	return resolved, nil
}

// RestorePathsForDisk converte caminhos lidos do banco em caminhos absolutos testáveis no disco físico.
func RestorePathsForDisk(tagName string, paths []string) ([]string, error) {
	meta, err := storage.GetTagMeta(tagName)
	if err != nil {
		return nil, err
	}

	if meta.Type == storage.TagTypeGit {
		gitRoot := meta.GitRoot

		// Fallback de retrocompatibilidade para tags antigas
		if gitRoot == "" {
			if !vcs.IsInsideRepo() || vcs.GetRepoID() != meta.RepoID {
				displayName := meta.RepoName
				if displayName == "" {
					displayName = meta.RepoID
				}
				return nil, fmt.Errorf("esta tag não possui a raiz do Git salva. Execute este comando dentro do repositório [%s] uma vez para que ela seja lida", displayName)
			}
			gitRoot = vcs.GetRoot()
		}

		var absPaths []string
		for _, p := range paths {
			absPaths = append(absPaths, filepath.ToSlash(filepath.Join(gitRoot, p)))
		}
		return absPaths, nil
	}

	return paths, nil
}

// ExpandPathsToFiles expande diretórios iterando recursivamente, respeitando a denylist
func ExpandPathsToFiles(paths []string, ignored map[string]bool) []string {
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
