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

var destFlag string

var exportCmd = &cobra.Command{
	Use:   "export <nome da tag>",
	Short: "Exporta os arquivos monitorados para um diretório usando concorrência",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tagName := args[0]

		files := getTagFiles(tagName)
		if len(files) == 0 {
			fmt.Printf("A tag '%s' não possui arquivos rastreados ou não existe.\n", tagName)
			os.Exit(1)
		}

		if err := os.MkdirAll(destFlag, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar destino: %v\n", err)
			os.Exit(1)
		}

		// Calcula a raiz comum para não copiar árvores de diretórios inteiras do SO
		basePrefix := getCommonPrefix(files)

		numWorkers := runtime.NumCPU()
		fmt.Printf("Iniciando exportação de %d arquivo(s) com %d worker(s)...\n", len(files), numWorkers)

		// Channel para distribuir o trabalho (Buffered)
		jobs := make(chan string, len(files))
		var wg sync.WaitGroup

		// 1. Inicializa o Pool de Goroutines
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go worker(jobs, &wg, basePrefix, destFlag)
		}

		// 2. Alimenta os channels
		for _, file := range files {
			jobs <- file
		}
		close(jobs) // Sinaliza aos workers que não há mais jobs

		// 3. Bloqueia até que todos os workers terminem o I/O
		wg.Wait()
		fmt.Printf("\nSucesso! Arquivos exportados para a pasta '%s'.\n", destFlag)
	},
}

func init() {
	exportCmd.Flags().StringVarP(&destFlag, "dest", "d", "", "Diretório de destino (obrigatório)")
	exportCmd.MarkFlagRequired("dest")
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
		// Remove o prefixo para obter um caminho relativo ao destino
		relPath := strings.TrimPrefix(path, basePrefix)
		relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

		// Fallback caso seja um único arquivo na raiz sem diretório pai claro
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

	// Garante que a árvore de subpastas da tag exista no destino
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
