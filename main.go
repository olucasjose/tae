package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Uso incorreto.\nExemplos:\n  %s HEAD~5 HEAD\n  %s main origin/main\n", filepath.Base(os.Args[0]), filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	commit1 := os.Args[1]
	commit2 := os.Args[2]

	fmt.Printf("Comparando %s -> %s\n\n", commit1, commit2)

	files := getChangedFiles(commit1, commit2)
	createZip(files)
}

func getChangedFiles(c1, c2 string) []string {
	cmd := exec.Command("git", "diff", "--name-status", c1, c2)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao executar git diff. Certifique-se de que está em um repositório Git válido.\n%s\n", stderr.String())
		os.Exit(1)
	}

	var filesToZip []string
	lines := strings.Split(out.String(), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		statusCode := strings.ToUpper(parts[0])
		statusChar := statusCode[0]

		var filePath string
		isRename := false

		// Filtra A (Added), M (Modified) e R (Renamed)
		if (statusChar == 'A' || statusChar == 'M') && len(parts) >= 2 {
			filePath = parts[1]
		} else if statusChar == 'R' && len(parts) >= 3 {
			filePath = parts[2]
			isRename = true
		} else {
			continue
		}

		// Checa se é um arquivo válido e existente antes de engatilhar para o zip
		info, err := os.Stat(filePath)
		if err == nil && !info.IsDir() {
			filesToZip = append(filesToZip, filePath)
			if isRename {
				fmt.Printf("  R: %s (renomeado)\n", filePath)
			} else {
				fmt.Printf("  %c: %s\n", statusChar, filePath)
			}
		}
	}

	return filesToZip
}

func createZip(files []string) {
	if len(files) == 0 {
		fmt.Println("\nNenhum arquivo encontrado para compactar no disco.")
		os.Exit(0)
	}

	fmt.Printf("\n%d arquivo(s) para compactar.\n", len(files))

	timestamp := time.Now().Format("20060102_150405")
	zipName := fmt.Sprintf("git_changes_%s.zip", timestamp)

	fmt.Printf("Criando %s...\n", zipName)

	zipFile, err := os.Create(zipName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nErro de I/O ao criar o arquivo zip: %v\n", err)
		os.Exit(1)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for _, file := range files {
		if err := addFileToZip(archive, file); err != nil {
			fmt.Fprintf(os.Stderr, "Aviso: falha ao adicionar %s ao zip: %v\n", file, err)
		}
	}

	fmt.Printf("\nSucesso! %s criado com %d arquivo(s).\n", zipName, len(files))
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	// O defer aqui é crucial para não esgotar os file descriptors em loops longos
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	
	// Preserva o caminho original (pastas/subpastas) dentro do zip
	header.Name = filename 
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, fileToZip)
	return err
}