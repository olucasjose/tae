package storage

import (
	"fmt"
	"path/filepath"

	"go.etcd.io/bbolt"
)

// TrackPath converte o caminho para absoluto e o insere no sub-bucket do projeto
func TrackPath(projectName, targetPath string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("caminho inválido: %w", err)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		projBucket := tx.Bucket([]byte(BucketProjects))
		if projBucket.Get([]byte(projectName)) == nil {
			return fmt.Errorf("projeto '%s' não existe. Crie-o primeiro com 'spycode project create'", projectName)
		}

		filesBucket := tx.Bucket([]byte(BucketFiles))
		
		// Cria um bucket aninhado com o nome do projeto
		projFiles, err := filesBucket.CreateBucketIfNotExists([]byte(projectName))
		if err != nil {
			return fmt.Errorf("falha ao estruturar bucket do projeto: %w", err)
		}

		// O valor vazio ("1") é um placeholder. No futuro, armazenaremos hashes aqui.
		return projFiles.Put([]byte(absPath), []byte("1"))
	})
}

// UntrackPath remove um caminho do sub-bucket do projeto
func UntrackPath(projectName, targetPath string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("caminho inválido: %w", err)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		filesBucket := tx.Bucket([]byte(BucketFiles))
		if filesBucket == nil {
			return nil
		}
		projFiles := filesBucket.Bucket([]byte(projectName))
		if projFiles == nil {
			return fmt.Errorf("projeto '%s' não possui arquivos rastreados ou não existe", projectName)
		}

		return projFiles.Delete([]byte(absPath))
	})
}
