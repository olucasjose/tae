// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
)

// RenameTag transfere atomicamente os índices de uma tag para um novo nome.
func RenameTag(oldName, newName string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRow("SELECT 1 FROM tags WHERE name = ?", newName).Scan(&exists)
	if err == nil {
		return fmt.Errorf("a tag destino '%s' já existe. Operação abortada", newName)
	} else if err != sql.ErrNoRows {
		return err
	}

	res, err := tx.Exec("UPDATE tags SET name = ? WHERE name = ?", newName, oldName)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("a tag origem '%s' não existe", oldName)
	}

	return tx.Commit()
}
