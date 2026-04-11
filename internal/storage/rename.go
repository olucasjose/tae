// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
	"go.etcd.io/bbolt"
)

// RenameTag transfere atomicamente os índices de uma tag para um novo nome.
func RenameTag(oldName, newName string) error {
	db, err := Open()
	if err != nil {
		return err
	}
	defer db.Close()

	return db.Update(func(tx *bbolt.Tx) error {
		tagsBucket := tx.Bucket([]byte(BucketTags))
		filesBucket := tx.Bucket([]byte(BucketFiles))
		ignoredBucket := tx.Bucket([]byte(BucketIgnored))

		// Validações Fail-Fast
		if tagsBucket.Get([]byte(oldName)) == nil {
			return fmt.Errorf("a tag origem '%s' não existe", oldName)
		}
		if tagsBucket.Get([]byte(newName)) != nil {
			return fmt.Errorf("a tag destino '%s' já existe. Operação abortada", newName)
		}

		// 1. Atualiza registro raiz (BucketTags)
		val := tagsBucket.Get([]byte(oldName))
		if err := tagsBucket.Put([]byte(newName), val); err != nil {
			return err
		}
		if err := tagsBucket.Delete([]byte(oldName)); err != nil {
			return err
		}

		// Função auxiliar para transferir os sub-buckets (Files e Ignored)
		transferSubBucket := func(parent *bbolt.Bucket) error {
			if parent == nil {
				return nil
			}
			oldBucket := parent.Bucket([]byte(oldName))
			if oldBucket == nil {
				return nil
			}
			newBucket, err := parent.CreateBucket([]byte(newName))
			if err != nil {
				return err
			}
			err = oldBucket.ForEach(func(k, v []byte) error {
				return newBucket.Put(k, v)
			})
			if err != nil {
				return err
			}
			return parent.DeleteBucket([]byte(oldName))
		}

		// 2. Transfere metadados de rastreamento
		if err := transferSubBucket(filesBucket); err != nil {
			return fmt.Errorf("falha ao transferir arquivos rastreados: %w", err)
		}

		// 3. Transfere Exclusion Index
		if err := transferSubBucket(ignoredBucket); err != nil {
			return fmt.Errorf("falha ao transferir a denylist: %w", err)
		}

		return nil
	})
}
