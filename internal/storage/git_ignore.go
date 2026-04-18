// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
)

// GitIgnorePaths adiciona os caminhos alvo (já processados como relativos à raiz do git)
// na denylist persistente do repositório específico.
func GitIgnorePaths(repoID string, targets []string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO git_ignored (repo_id, path) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, path := range targets {
		if _, err := stmt.Exec(repoID, path); err != nil {
			return fmt.Errorf("erro ao inserir na denylist git: %w", err)
		}
	}

	return tx.Commit()
}

// GetGitIgnoredPaths retorna um hash map rápido contendo a denylist do repositório.
func GetGitIgnoredPaths(repoID string) (map[string]bool, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT path FROM git_ignored WHERE repo_id = ?", repoID)
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

// UnignoreGitPaths remove caminhos da denylist do repositório.
func UnignoreGitPaths(repoID string, targets []string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("DELETE FROM git_ignored WHERE repo_id = ? AND path = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, path := range targets {
		if _, err := stmt.Exec(repoID, path); err != nil {
			return fmt.Errorf("erro ao remover caminho da denylist: %w", err)
		}
	}

	return tx.Commit()
}
