// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
	"strings"

	"go.etcd.io/bbolt"
)

// CreateTags cadastra múltiplas tags no banco de forma atômica.
func CreateTags(tags []string, meta TagMeta) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketTags))
		
		for _, tagName := range tags {
			if b.Get([]byte(tagName)) != nil {
				return fmt.Errorf("a tag '%s' já existe. Operação abortada", tagName)
			}
		}
		
		encodedMeta := EncodeTagMeta(meta)

		for _, tagName := range tags {
			if err := b.Put([]byte(tagName), encodedMeta); err != nil {
				return fmt.Errorf("erro ao gravar tag '%s': %w", tagName, err)
			}
		}
		return nil
	})
}

// DeleteTags remove as tags e seus respectivos buckets de rastreamento em uma transação única.
func DeleteTags(tags []string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		tagsBucket := tx.Bucket([]byte(BucketTags))
		filesBucket := tx.Bucket([]byte(BucketFiles))

		var missingTags []string

		// Validação Fail-Fast Acumulativa
		for _, tagName := range tags {
			if tagsBucket.Get([]byte(tagName)) == nil {
				missingTags = append(missingTags, tagName)
			}
		}

		if len(missingTags) > 0 {
			return fmt.Errorf("as seguintes tags não existem: %s. Operação abortada", strings.Join(missingTags, ", "))
		}

		// Execução destrutiva
		for _, tagName := range tags {
			if err := tagsBucket.Delete([]byte(tagName)); err != nil {
				return fmt.Errorf("falha ao remover referência da tag '%s': %w", tagName, err)
			}

			if filesBucket.Bucket([]byte(tagName)) != nil {
				if err := filesBucket.DeleteBucket([]byte(tagName)); err != nil {
					return fmt.Errorf("falha ao remover rastreamento da tag '%s': %w", tagName, err)
				}
			}
		}
		return nil
	})
}

// GetFilesByTag retorna a lista plana de todos os caminhos rastreados por uma tag.
func GetFilesByTag(tagName string) ([]string, error) {
	db, err := Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var files []string
	err = db.View(func(tx *bbolt.Tx) error {
		filesBucket := tx.Bucket([]byte(BucketFiles))
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
	return files, err
}

// GetAllTagsWithMeta recupera o dicionário completo de tags e seus metadados para varreduras de listagem.
func GetAllTagsWithMeta() (map[string]TagMeta, error) {
	db, err := Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	tags := make(map[string]TagMeta)
	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketTags))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			tags[string(k)] = ParseTagMeta(v)
			return nil
		})
	})
	return tags, err
}
