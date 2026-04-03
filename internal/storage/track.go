package storage

import (
	"fmt"
	"path/filepath"

	"go.etcd.io/bbolt"
)

// TrackPath converte o caminho para absoluto e o insere no sub-bucket da tag
func TrackPath(tagName, targetPath string) error {
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
		projBucket := tx.Bucket([]byte(BucketTags))
		if projBucket.Get([]byte(tagName)) == nil {
			return fmt.Errorf("tag '%s' não existe. Crie-a primeiro com 'tae tag create'", tagName)
		}

		filesBucket := tx.Bucket([]byte(BucketFiles))
		
		// Cria um bucket aninhado com o nome da tag
		projFiles, err := filesBucket.CreateBucketIfNotExists([]byte(tagName))
		if err != nil {
			return fmt.Errorf("falha ao estruturar bucket da tag: %w", err)
		}

		// O valor vazio ("1") é um placeholder. No futuro, armazenaremos hashes aqui.
		return projFiles.Put([]byte(absPath), []byte("1"))
	})
}

// UntrackPath remove um caminho do sub-bucket da tag
func UntrackPath(tagName, targetPath string) error {
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
		projFiles := filesBucket.Bucket([]byte(tagName))
		if projFiles == nil {
			return fmt.Errorf("tag '%s' não possui arquivos rastreados ou não existe", tagName)
		}

		return projFiles.Delete([]byte(absPath))
	})
}
