package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"tae/internal/storage"

	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var exportCmd = &cobra.Command{
	Use:   "export <nome da tag> <destino>",
	Short: "Exporta os arquivos monitorados para um diretório de destino",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[0]
		destPath := args[1]

		files := getTagFiles(tagName)
		if len(files) == 0 {
			fmt.Printf("A tag '%s' não possui arquivos rastreados ou não existe.\n", tagName)
			os.Exit(1)
		}

		if err := os.MkdirAll(destPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar destino: %v\n", err)
			os.Exit(1)
		}

		basePrefix := getCommonPrefix(files)
		numWorkers := runtime.NumCPU()
		fmt.Printf("Iniciando exportação de %d arquivo(s) para '%s' com %d worker(s)...\n", len(files), destPath, numWorkers)

		jobs := make(chan string, len(files))
		var wg sync.WaitGroup

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go worker(jobs, &wg, basePrefix, destPath)
		}

		for _, file := range files {
			jobs <- file
		}
		close(jobs)

		wg.Wait()
		fmt.Printf("\nSucesso! Arquivos exportados para a pasta '%s'.\n", destPath)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func getTagFiles(tagName string) []string {
	var files []string
	db, err := storage.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao abrir banco: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	db.View(func(tx *bbolt.Tx) error {
		filesBucket := tx.Bucket([]byte(storage.BucketFiles))
		if filesBucket == nil {
			return nil
		}
		projFiles := filesBucket.Bucket([]byte(tagName))
		if projFiles == nil {
			return nil
		}

		return projFiles.ForEach(func(k, v []byte) error {
			files = append(files, string(k))
			return nil
		})
	})
	return files
}

func worker(jobs <-chan string, wg *sync.WaitGroup, basePrefix, dest string) {
	defer wg.Done()
	for path := range jobs {
		relPath := strings.TrimPrefix(path, basePrefix)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

		if relPath == "" {
			relPath = filepath.Base(path)
		}

		targetPath := filepath.Join(dest, relPath)

		if err := copyFile(path, targetPath); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao copiar %s: %v\n", path, err)
		} else {
			fmt.Printf("  -> %s\n", targetPath)
		}
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
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
