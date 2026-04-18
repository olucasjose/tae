// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
	"strings"
)

// CreateTags cadastra múltiplas tags no banco de forma atômica.
func CreateTags(tags []string, meta TagMeta) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	// Removido: defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Validação Fail-Fast
	for _, tagName := range tags {
		var exists int
		err := tx.QueryRow("SELECT 1 FROM tags WHERE name = ?", tagName).Scan(&exists)
		if err != sql.ErrNoRows && err != nil {
			return fmt.Errorf("erro ao verificar existência da tag '%s': %w", tagName, err)
		}
		if exists == 1 {
			return fmt.Errorf("a tag '%s' já existe. Operação abortada", tagName)
		}
	}

	stmt, err := tx.Prepare("INSERT INTO tags (name, type, repo_id, repo_name, git_root) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tagName := range tags {
		_, err := stmt.Exec(tagName, meta.Type, meta.RepoID, meta.RepoName, meta.GitRoot)
		if err != nil {
			return fmt.Errorf("erro ao gravar tag '%s': %w", tagName, err)
		}
	}

	return tx.Commit()
}

// DeleteTags remove as tags e aciona em cascata a limpeza de seus arquivos rastreados.
func DeleteTags(tags []string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var missingTags []string

	// Validação Fail-Fast Acumulativa
	for _, tagName := range tags {
		var exists int
		err := tx.QueryRow("SELECT 1 FROM tags WHERE name = ?", tagName).Scan(&exists)
		if err == sql.ErrNoRows {
			missingTags = append(missingTags, tagName)
		} else if err != nil {
			return err
		}
	}

	if len(missingTags) > 0 {
		return fmt.Errorf("as seguintes tags não existem: %s. Operação abortada", strings.Join(missingTags, ", "))
	}

	// Execução destrutiva
	stmt, err := tx.Prepare("DELETE FROM tags WHERE name = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tagName := range tags {
		if _, err := stmt.Exec(tagName); err != nil {
			return fmt.Errorf("falha ao remover referência da tag '%s': %w", tagName, err)
		}
	}

	return tx.Commit()
}

// GetFilesByTag retorna a lista plana de todos os caminhos rastreados por uma tag.
func GetFilesByTag(tagName string) ([]string, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT path FROM files_tracked WHERE tag_name = ?", tagName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		files = append(files, path)
	}
	return files, rows.Err()
}

// GetAllTagsWithMeta recupera o dicionário completo de tags e seus metadados para varreduras de listagem.
func GetAllTagsWithMeta() (map[string]TagMeta, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT name, type, repo_id, repo_name, git_root FROM tags")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make(map[string]TagMeta)
	for rows.Next() {
		var name string
		var meta TagMeta
		var repoID, repoName, gitRoot sql.NullString

		if err := rows.Scan(&name, &meta.Type, &repoID, &repoName, &gitRoot); err != nil {
			return nil, err
		}

		if repoID.Valid {
			meta.RepoID = repoID.String
		}
		if repoName.Valid {
			meta.RepoName = repoName.String
		}
		if gitRoot.Valid {
			meta.GitRoot = gitRoot.String
		}

		tags[name] = meta
	}
	return tags, rows.Err()
}
