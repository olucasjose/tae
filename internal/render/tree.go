// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package render

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"tae/internal/filter"
)

// GetCommonPrefix centraliza a descoberta de raiz comum.
func GetCommonPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	if len(paths) == 1 {
		return filepath.Dir(paths[0]) + string(filepath.Separator)
	}
	prefix := strings.Split(paths[0], string(filepath.Separator))
	for _, p := range paths[1:] {
		parts := strings.Split(p, string(filepath.Separator))
		limit := len(prefix)
		if len(parts) < limit {
			limit = len(parts)
		}
		var i int
		for i = 0; i < limit; i++ {
			if prefix[i] != parts[i] {
				break
			}
		}
		prefix = prefix[:i]
	}
	return strings.Join(prefix, string(filepath.Separator)) + string(filepath.Separator)
}

// TreeNode é focado apenas na renderização visual (diferente do Node do grouper).
type TreeNode struct {
	Name     string
	IsDir    bool
	Children map[string]*TreeNode
}

// BuildVisualTree mapeia a lista plana em uma hierarquia visual.
func BuildVisualTree(files []string, basePrefix string) *TreeNode {
	root := &TreeNode{Name: basePrefix, IsDir: true, Children: make(map[string]*TreeNode)}

	for _, f := range files {
		relPath := strings.TrimPrefix(f, basePrefix)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		if relPath == "" {
			relPath = filepath.Base(f)
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		curr := root

		for i, part := range parts {
			isLast := i == len(parts)-1
			if curr.Children[part] == nil {
				curr.Children[part] = &TreeNode{
					Name:     part,
					IsDir:    !isLast,
					Children: make(map[string]*TreeNode),
				}
			}
			curr = curr.Children[part]
		}
	}
	return root
}

// PrintTree imprime a árvore com as ramificações de terminal em um io.Writer qualquer.
func PrintTree(w io.Writer, node *TreeNode, prefix string, depth, maxDepth int, ignorePatterns []string) {
	if maxDepth > 0 && depth > maxDepth {
		return
	}

	var keys []string
	for k, child := range node.Children {
		if !filter.MatchPattern(child.Name, ignorePatterns) {
			keys = append(keys, k)
		}
	}

	// Ordena: pastas primeiro, depois arquivos alfabeticamente
	sort.Slice(keys, func(i, j int) bool {
		childI := node.Children[keys[i]]
		childJ := node.Children[keys[j]]
		if childI.IsDir != childJ.IsDir {
			return childI.IsDir
		}
		return keys[i] < keys[j]
	})

	for i, k := range keys {
		child := node.Children[k]
		isLast := i == len(keys)-1

		connector := "├── "
		if isLast {
			connector = "└── "
		}

		fmt.Fprintf(w, "%s%s%s\n", prefix, connector, child.Name)

		if child.IsDir {
			extension := "│   "
			if isLast {
				extension = "    "
			}
			PrintTree(w, child, prefix+extension, depth+1, maxDepth, ignorePatterns)
		}
	}
}
