// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
	"path/filepath"

	"go.etcd.io/bbolt"
)

// IgnorePaths move alvos explícitos do fluxo de rastreamento para a blacklist da tag.
func IgnorePaths(tagName string, targets []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	// Resolve caminhos absolutos antes de travar o banco
	var absTargets []string
	for _, t := range targets {
		absPath, err := filepath.Abs(t)
		if err != nil {
			return fmt.Errorf("caminho inválido '%s': %w", t, err)
		}
		absTargets = append(absTargets, absPath)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		tagsBucket := tx.Bucket([]byte(BucketTags))
		if tagsBucket.Get([]byte(tagName)) == nil {
			// Mantém a consistência de auto-criação da tag
			if err := tagsBucket.Put([]byte(tagName), []byte("{}")); err != nil {
				return fmt.Errorf("falha ao inicializar tag: %w", err)
			}
		}

		filesBucket := tx.Bucket([]byte(BucketFiles))
		ignoredBucket := tx.Bucket([]byte(BucketIgnored))

		projFiles, err := filesBucket.CreateBucketIfNotExists([]byte(tagName))
		if err != nil {
			return err
		}

		projIgnored, err := ignoredBucket.CreateBucketIfNotExists([]byte(tagName))
		if err != nil {
			return err
		}

		for _, absPath := range absTargets {
			// Remove do mapeamento principal (se estiver lá)
			if projFiles.Get([]byte(absPath)) != nil {
				if err := projFiles.Delete([]byte(absPath)); err != nil {
					return err
				}
			}
			// Indexa no Exclusion Index
			if err := projIgnored.Put([]byte(absPath), []byte("1")); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetIgnoredPaths carrega o Exclusion Index da tag em um mapa para lookup O(1) em tempo de execução.
func GetIgnoredPaths(tagName string) (map[string]bool, error) {
	db, err := Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ignored := make(map[string]bool)
	err = db.View(func(tx *bbolt.Tx) error {
		ignoredBucket := tx.Bucket([]byte(BucketIgnored))
		if ignoredBucket == nil {
			return nil
		}
		projIgnored := ignoredBucket.Bucket([]byte(tagName))
		if projIgnored == nil {
			return nil
		}
		return projIgnored.ForEach(func(k, v []byte) error {
			ignored[string(k)] = true
			return nil
		})
	})
	return ignored, err
}

// UnignorePaths remove alvos da blacklist, devolvendo-os ao fluxo de herança original.
func UnignorePaths(tagName string, targets []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	var absTargets []string
	for _, t := range targets {
		absPath, err := filepath.Abs(t)
		if err != nil {
			return fmt.Errorf("caminho inválido '%s': %w", t, err)
		}
		absTargets = append(absTargets, absPath)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		ignoredBucket := tx.Bucket([]byte(BucketIgnored))
		if ignoredBucket == nil {
			return nil // Se não existe bucket, não há o que des-ignorar
		}

		projIgnored := ignoredBucket.Bucket([]byte(tagName))
		if projIgnored == nil {
			return nil
		}

		for _, absPath := range absTargets {
			if err := projIgnored.Delete([]byte(absPath)); err != nil {
				return err
			}
		}
		return nil
	})
}
