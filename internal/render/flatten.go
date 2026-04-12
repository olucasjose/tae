// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package render

import (
	"path/filepath"
	"sort"
	"strings"
)

// ResolveFlattenNames gera um mapa de caminhos originais para caminhos nivelados (sem pastas).
// Resolve colisões priorizando o arquivo mais raso com o nome original e prefixando os mais profundos.
func ResolveFlattenNames(files []string, basePrefix string) map[string]string {
	grouped := make(map[string][]string)
	for _, f := range files {
		base := filepath.Base(f)
		grouped[base] = append(grouped[base], f)
	}

	flattenMap := make(map[string]string)

	for base, group := range grouped {
		if len(group) == 1 {
			flattenMap[group[0]] = base
			continue
		}

		// Ordena por profundidade. Normalizamos para ToSlash garantindo compatibilidade OS/Git
		sort.Slice(group, func(i, j int) bool {
			relI := filepath.ToSlash(strings.TrimPrefix(group[i], basePrefix))
			relJ := filepath.ToSlash(strings.TrimPrefix(group[j], basePrefix))
			depthI := strings.Count(relI, "/")
			depthJ := strings.Count(relJ, "/")

			if depthI != depthJ {
				return depthI < depthJ
			}
			return relI < relJ
		})

		for i, f := range group {
			if i == 0 {
				flattenMap[f] = base
			} else {
				relPath := filepath.ToSlash(strings.TrimPrefix(f, basePrefix))
				relPath = strings.TrimPrefix(relPath, "/")
				dir := filepath.Dir(relPath)
				if dir == "." {
					flattenMap[f] = base
					continue
				}
				// Troca separadores de diretório por hífen
				prefix := strings.ReplaceAll(dir, "/", "-")
				flattenMap[f] = prefix + "-" + base
			}
		}
	}

	return flattenMap
}
