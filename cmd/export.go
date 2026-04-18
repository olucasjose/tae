// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"tae/internal/grouper"
	"tae/internal/render"
	"tae/internal/storage"
	"tae/internal/exporter"

	"github.com/spf13/cobra"
)

var (
	exportZip     bool
	exportLimit   int
	exportMerge   bool
	exportFlatten bool
	exportQuiet   bool
)

var exportCmd = &cobra.Command{
	Use:   "export <nome da tag> <destino>",
	Short: "Exporta os arquivos e diretórios monitorados para um destino",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			tags, _ := storage.GetAllTags()
			return tags, cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 {
			return nil, cobra.ShellCompDirectiveFilterDirs
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tagName := args[0]
		destPath := args[1]

		rawFiles, err := storage.GetFilesByTag(tagName)
		if err != nil {
			return fmt.Errorf("erro ao buscar rastreamento da tag: %w", err)
		}
		if len(rawFiles) == 0 {
			return fmt.Errorf("a tag '%s' não possui alvos rastreados ou não existe", tagName)
		}

		resolvedFiles, err := restorePathsForDisk(tagName, rawFiles)
		if err != nil {
			return fmt.Errorf("erro de escopo estrutural: %w", err)
		}

		ignoredMap, err := storage.GetIgnoredPaths(tagName)
		if err != nil {
			fmt.Printf("Aviso: Falha ao carregar Exclusion Index: %v\n", err)
		}

		restoredIgnored := make(map[string]bool)
		var igPaths []string
		for p := range ignoredMap {
			igPaths = append(igPaths, p)
		}
		if resIgPaths, err := restorePathsForDisk(tagName, igPaths); err == nil {
			for _, p := range resIgPaths {
				restoredIgnored[p] = true
			}
		}

		files := expandPathsToFiles(resolvedFiles, restoredIgnored)
		if len(files) == 0 {
			return fmt.Errorf("nenhum arquivo válido encontrado (possivelmente todos foram ignorados)")
		}

		if err := os.MkdirAll(destPath, 0755); err != nil {
			return fmt.Errorf("erro ao criar destino: %w", err)
		}

		basePrefix := render.GetCommonPrefix(files)
		numWorkers := runtime.NumCPU()

		var flattenMap map[string]string
		if exportFlatten {
			flattenMap = render.ResolveFlattenNames(files, basePrefix)
		}

		opts := exporter.ExportOptions{
			DestDir:    destPath,
			BasePrefix: basePrefix,
			FlattenMap: flattenMap,
			Quiet:      exportQuiet,
		}

		if exportZip {
			chunks := grouper.GroupFiles(files, exportLimit, tagName, exportMerge)
			fmt.Printf("Iniciando exportação em ZIP. %d arquivo(s) expandido(s) divididos em %d lote(s) para '%s'...\n", len(files), len(chunks), destPath)
			if !exportQuiet {
				fmt.Printf("[Raiz Comum: %s]\n\n", basePrefix)
			}
			exporter.ExportZip(chunks, numWorkers, opts)
			fmt.Printf("\nSucesso! %d arquivo(s) zip gerado(s) na pasta '%s'.\n", len(chunks), destPath)
		} else {
			fmt.Printf("Iniciando exportação plana de %d arquivo(s) para '%s' com %d worker(s)....\n", len(files), destPath, numWorkers)
			exporter.ExportFlat(files, numWorkers, opts)
			fmt.Printf("\nSucesso! Arquivos exportados para a pasta '%s'.\n", destPath)
		}
		return nil
	},
}

// Nota da Gina: Funções auxiliares mantidas intactas, ajuste apenas o os.Stderr interno para print para evitar side-effects
func init() {
	exportCmd.Flags().BoolVarP(&exportZip, "zip", "z", false, "Exporta e compacta os arquivos em formato .zip")
	exportCmd.Flags().IntVarP(&exportLimit, "limit", "l", 0, "Teto máximo de arquivos por zip (requer -z)")
	exportCmd.Flags().BoolVarP(&exportMerge, "merge", "m", false, "Mescla zips subpopulados sequencialmente mantendo o limite (requer -z e -l)")
	exportCmd.Flags().BoolVarP(&exportFlatten, "flatten", "f", false, "Exporta todos os arquivos no mesmo nível (sem pastas), resolvendo colisões de nomes")
	exportCmd.Flags().BoolVarP(&exportQuiet, "quiet", "q", false, "Oculta a listagem individual dos arquivos no console")
	rootCmd.AddCommand(exportCmd)
}

func expandPathsToFiles(paths []string, ignored map[string]bool) []string {
	uniqueFiles := make(map[string]bool)
	var expanded []string

	for _, p := range paths {
		if ignored[p] { continue }
		info, err := os.Stat(p)
		if err != nil { continue }
		if !info.IsDir() {
			if !uniqueFiles[p] {
				uniqueFiles[p] = true
				expanded = append(expanded, p)
			}
			continue
		}
		filepath.Walk(p, func(path string, f os.FileInfo, err error) error {
			if err != nil { return nil }
			if ignored[path] {
				if f.IsDir() { return filepath.SkipDir }
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
