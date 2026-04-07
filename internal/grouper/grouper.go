package grouper

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type Node struct {
	Name       string
	Files      []string
	SubDirs    map[string]*Node
	totalCount int
}

func (n *Node) TotalFiles() int {
	if n.totalCount > 0 {
		return n.totalCount
	}
	count := len(n.Files)
	for _, d := range n.SubDirs {
		count += d.TotalFiles()
	}
	n.totalCount = count
	return count
}

type Chunk struct {
	Prefix string
	Files  []string
}

type ExportChunk struct {
	ZipName string
	Files   []string
}

// GroupFiles gera os lotes fatiados. Se merge=true, aciona a fusão de blocos subpopulados.
func GroupFiles(files []string, limit int, baseName string, merge bool) []ExportChunk {
	if limit <= 0 || len(files) <= limit {
		return []ExportChunk{{ZipName: fmt.Sprintf("%s.zip", baseName), Files: files}}
	}

	basePrefix := getCommonPrefix(files)
	root := buildTree(files, basePrefix)

	var chunks []*Chunk

	var allocate func(node *Node, prefix string, activeChunk *Chunk) *Chunk
	allocate = func(node *Node, prefix string, activeChunk *Chunk) *Chunk {
		sort.Strings(node.Files)
		
		for _, f := range node.Files {
			if len(activeChunk.Files) >= limit {
				chunks = append(chunks, activeChunk)
				activeChunk = &Chunk{Prefix: prefix}
			}
			activeChunk.Files = append(activeChunk.Files, f)
		}

		var dirNames []string
		for d := range node.SubDirs {
			dirNames = append(dirNames, d)
		}
		sort.Strings(dirNames)

		for _, dName := range dirNames {
			subNode := node.SubDirs[dName]
			
			nodePrefix := prefix
			if nodePrefix == baseName {
				nodePrefix = fmt.Sprintf("%s_%s", baseName, dName)
			}

			if len(activeChunk.Files)+subNode.TotalFiles() <= limit {
				activeChunk = allocate(subNode, prefix, activeChunk)
			} else {
				childChunk := &Chunk{Prefix: nodePrefix}
				finalChildChunk := allocate(subNode, nodePrefix, childChunk)
				
				if len(finalChildChunk.Files) > 0 {
					chunks = append(chunks, finalChildChunk)
				}
			}
		}
		return activeChunk
	}

	finalChunk := allocate(root, baseName, &Chunk{Prefix: baseName})
	if len(finalChunk.Files) > 0 {
		chunks = append(chunks, finalChunk)
	}

	exports := formatChunkNames(chunks)

	if merge {
		return mergeBlocks(exports, limit, baseName)
	}

	return exports
}

// mergeBlocks aplica a heurística Next-Fit em pós-processamento preservando limites e nomes
func mergeBlocks(chunks []ExportChunk, limit int, baseName string) []ExportChunk {
	if len(chunks) <= 1 {
		return chunks
	}

	var merged []ExportChunk
	current := chunks[0]

	stripZip := func(s string) string { return strings.TrimSuffix(s, ".zip") }

	for i := 1; i < len(chunks); i++ {
		next := chunks[i]
		
		if len(current.Files)+len(next.Files) <= limit {
			current.Files = append(current.Files, next.Files...)
			
			cName := stripZip(current.ZipName)
			nName := stripZip(next.ZipName)
			
			// Sanitiza o prefixo base para o nome final não ficar colossal
			cleanNext := strings.TrimPrefix(nName, baseName+"_")
			if cleanNext == baseName {
				cleanNext = "root"
			}
			current.ZipName = fmt.Sprintf("%s+%s.zip", cName, cleanNext)
		} else {
			merged = append(merged, current)
			current = next
		}
	}
	merged = append(merged, current)
	return merged
}

func formatChunkNames(chunks []*Chunk) []ExportChunk {
	prefixCounts := make(map[string]int)
	prefixTotals := make(map[string]int)

	for _, c := range chunks {
		prefixTotals[c.Prefix]++
	}

	var exports []ExportChunk
	for _, c := range chunks {
		prefixCounts[c.Prefix]++
		name := fmt.Sprintf("%s.zip", c.Prefix)
		
		if prefixTotals[c.Prefix] > 1 {
			name = fmt.Sprintf("%s_part%d.zip", c.Prefix, prefixCounts[c.Prefix])
		}
		
		exports = append(exports, ExportChunk{
			ZipName: name,
			Files:   c.Files,
		})
	}

	return exports
}

func buildTree(files []string, basePrefix string) *Node {
	root := &Node{SubDirs: make(map[string]*Node), Name: "root"}
	for _, f := range files {
		relPath := strings.TrimPrefix(f, basePrefix)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
		if relPath == "" {
			relPath = filepath.Base(f)
		}

		parts := strings.Split(relPath, string(filepath.Separator))
		curr := root
		
		for i := 0; i < len(parts)-1; i++ {
			dirName := parts[i]
			if curr.SubDirs[dirName] == nil {
				curr.SubDirs[dirName] = &Node{Name: dirName, SubDirs: make(map[string]*Node)}
			}
			curr = curr.SubDirs[dirName]
		}
		curr.Files = append(curr.Files, f)
	}
	return root
}

func getCommonPrefix(paths []string) string {
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
