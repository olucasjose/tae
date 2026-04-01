package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
)

const (
	BucketProjects = "Projects"
	BucketFiles    = "Files" // Preparação para a Fase 3
)

// getDBPath resolve o caminho seguro para o banco de dados global
func getDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("falha ao localizar diretório home: %w", err)
	}
	
	dir := filepath.Join(home, ".spycode")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório base: %w", err)
	}
	
	return filepath.Join(dir, "spycode.db"), nil
}

// Open inicia a conexão com o bbolt e garante a existência dos buckets
func Open() (*bbolt.DB, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, err
	}

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo .db: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(BucketProjects)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(BucketFiles)); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("falha ao criar buckets internos: %w", err)
	}

	return db, nil
}
