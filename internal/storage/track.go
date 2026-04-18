// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
)

// TrackPaths recebe caminhos já processados pelo resolver e insere no rastreamento.
func TrackPaths(tagName string, targets []string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Garante a existência da tag (cria como Local se não existir)
	var exists int
	err = tx.QueryRow("SELECT 1 FROM tags WHERE name = ?", tagName).Scan(&exists)
	if err == sql.ErrNoRows {
		_, err = tx.Exec("INSERT INTO tags (name, type) VALUES (?, ?)", tagName, TagTypeLocal)
		if err != nil {
			return fmt.Errorf("falha ao criar tag '%s' automaticamente: %w", tagName, err)
		}
	} else if err != nil {
		return err
	}

	delStmt, err := tx.Prepare("DELETE FROM files_ignored WHERE tag_name = ? AND path = ?")
	if err != nil {
		return err
	}
	defer delStmt.Close()

	insStmt, err := tx.Prepare("INSERT OR IGNORE INTO files_tracked (tag_name, path) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer insStmt.Close()

	for _, path := range targets {
		// Remove da denylist se estiver lá
		if _, err := delStmt.Exec(tagName, path); err != nil {
			return err
		}
		// Insere no rastreamento
		if _, err := insStmt.Exec(tagName, path); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UntrackPath remove um caminho pré-processado da tag.
func UntrackPath(tagName, targetPath string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	res, err := db.Exec("DELETE FROM files_tracked WHERE tag_name = ? AND path = ?", tagName, targetPath)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("o alvo '%s' não está rastreado diretamente na tag '%s'", targetPath, tagName)
	}

	return nil
}
