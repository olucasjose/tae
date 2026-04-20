// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"tae/internal/exporter"
	"tae/internal/fs"
	"tae/internal/grouper"
	"tae/internal/render"
	"tae/internal/storage"

	"github.com/spf13/cobra"
)

var (
	exportZip     bool
	exportLimit   int
	exportMerge   bool
	exportFlatten bool
	exportQuiet   bool
	exportTxt     bool
	exportSingle  bool
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
		if exportSingle && (exportZip || exportFlatten) {
			return fmt.Errorf("a flag --single-file (-s) é exclusiva e não pode ser usada simultaneamente com --zip ou --flatten")
		}
		if err != nil {
			return fmt.Errorf("erro ao buscar rastreamento da tag: %w", err)
		}
		if len(rawFiles) == 0 {
			return fmt.Errorf("a tag '%s' não possui alvos rastreados ou não existe", tagName)
		}

		resolvedFiles, err := fs.RestorePathsForDisk(tagName, rawFiles)
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
		if resIgPaths, err := fs.RestorePathsForDisk(tagName, igPaths); err == nil {
			for _, p := range resIgPaths {
				restoredIgnored[p] = true
			}
		}

		files := fs.ExpandPathsToFiles(resolvedFiles, restoredIgnored)
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
			AppendTxt:  exportTxt,
		}

		if exportSingle {
			timestamp := time.Now().Format("20060102_150405")
			fileName := fmt.Sprintf("%s_%s.txt", tagName, timestamp)
			fullPath := filepath.Join(destPath, fileName)

			fmt.Printf("Iniciando exportação Single-File (Single File Style). %d arquivo(s) expandido(s) para '%s'...\n", len(files), fullPath)
			if !exportQuiet {
				fmt.Printf("[Raiz Comum: %s]\n\n", basePrefix)
			}
			if err := exporter.ExportSingleFile(fullPath, files, opts); err != nil {
				return err
			}
			fmt.Printf("\nSucesso! Arquivo consolidado gerado em '%s'.\n", fullPath)
		} else if exportZip {
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

func init() {
	exportCmd.Flags().BoolVarP(&exportZip, "zip", "z", false, "Exporta e compacta os arquivos em formato .zip")
	exportCmd.Flags().IntVarP(&exportLimit, "limit", "l", 0, "Teto máximo de arquivos por zip (requer -z)")
	exportCmd.Flags().BoolVarP(&exportMerge, "merge", "m", false, "Mescla zips subpopulados sequencialmente mantendo o limite (requer -z e -l)")
	exportCmd.Flags().BoolVarP(&exportFlatten, "flatten", "f", false, "Exporta todos os arquivos no mesmo nível (sem pastas), resolvendo colisões de nomes")
	exportCmd.Flags().BoolVarP(&exportQuiet, "quiet", "q", false, "Oculta a listagem individual dos arquivos no console")
	exportCmd.Flags().BoolVar(&exportTxt, "txt", false, "Adiciona a extensão .txt a todos os arquivos exportados")
	exportCmd.Flags().BoolVarP(&exportSingle, "single-file", "s", false, "Exporta todos os arquivos em um único arquivo de texto plano (Single File Style)")
	rootCmd.AddCommand(exportCmd)
}
