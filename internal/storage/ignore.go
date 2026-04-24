// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
)

// IgnorePaths move alvos pré-processados para a denylist.
func IgnorePaths(tagName string, targets []string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Garante a existência da tag
	var exists int
	err = tx.QueryRow("SELECT 1 FROM tags WHERE name = ?", tagName).Scan(&exists)
	if err == sql.ErrNoRows {
		_, err = tx.Exec("INSERT INTO tags (name, type) VALUES (?, ?)", tagName, TagTypeLocal)
		if err != nil {
			return fmt.Errorf("falha ao inicializar tag '%s': %w", tagName, err)
		}
	} else if err != nil {
		return err
	}

	delStmt, err := tx.Prepare("DELETE FROM files_tracked WHERE tag_name = ? AND path = ?")
	if err != nil {
		return err
	}
	defer delStmt.Close()

	insStmt, err := tx.Prepare("INSERT OR IGNORE INTO files_ignored (tag_name, path) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer insStmt.Close()

	for _, path := range targets {
		if _, err := delStmt.Exec(tagName, path); err != nil {
			return err
		}
		if _, err := insStmt.Exec(tagName, path); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetIgnoredPaths carrega o Exclusion Index da tag em um mapa O(1).
func GetIgnoredPaths(tagName string) (map[string]bool, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT path FROM files_ignored WHERE tag_name = ?", tagName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ignored := make(map[string]bool)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		ignored[path] = true
	}
	return ignored, rows.Err()
}

// UnignorePaths remove alvos pré-processados da denylist.
func UnignorePaths(tagName string, targets []string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("DELETE FROM files_ignored WHERE tag_name = ? AND path = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, path := range targets {
		if _, err := stmt.Exec(tagName, path); err != nil {
			return err
		}
	}

	return tx.Commit()
}
