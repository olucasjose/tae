// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"fmt"
)

// GetTagRawKeys retorna as chaves do banco exatamente como estão gravadas.
func GetTagRawKeys(tagName string) (files []string, ignored []string, err error) {
	db, err := GetDB()
	if err != nil {
		return nil, nil, err
	}

	fRows, err := db.Query("SELECT path FROM files_tracked WHERE tag_name = ?", tagName)
	if err == nil {
		for fRows.Next() {
			var p string
			if fRows.Scan(&p) == nil {
				files = append(files, p)
			}
		}
		fRows.Close()
	}

	iRows, err := db.Query("SELECT path FROM files_ignored WHERE tag_name = ?", tagName)
	if err == nil {
		for iRows.Next() {
			var p string
			if iRows.Scan(&p) == nil {
				ignored = append(ignored, p)
			}
		}
		iRows.Close()
	}

	return files, ignored, nil
}

// RemoveKeysFromTag deleta chaves específicas dos buckets de uma tag (usado pelo prune).
func RemoveKeysFromTag(tagName string, filesToRemove, ignoredToRemove []string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if len(filesToRemove) > 0 {
		stmt, _ := tx.Prepare("DELETE FROM files_tracked WHERE tag_name = ? AND path = ?")
		for _, f := range filesToRemove {
			stmt.Exec(tagName, f)
		}
		stmt.Close()
	}
	if len(ignoredToRemove) > 0 {
		stmt, _ := tx.Prepare("DELETE FROM files_ignored WHERE tag_name = ? AND path = ?")
		for _, i := range ignoredToRemove {
			stmt.Exec(tagName, i)
		}
		stmt.Close()
	}

	return tx.Commit()
}

// UpdateTagScope reescreve os caminhos de uma tag resolvendo a troca de contexto (Local <-> Git).
func UpdateTagScope(tagName string, meta TagMeta, swapFiles, swapIgnored map[string]string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec("UPDATE tags SET type = ?, repo_id = ?, repo_name = ?, git_root = ? WHERE name = ?",
		meta.Type, meta.RepoID, meta.RepoName, meta.GitRoot, tagName)
	if err != nil {
		return err
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("a tag '%s' não existe", tagName)
	}

	if len(swapFiles) > 0 {
		delStmt, _ := tx.Prepare("DELETE FROM files_tracked WHERE tag_name = ? AND path = ?")
		insStmt, _ := tx.Prepare("INSERT INTO files_tracked (tag_name, path) VALUES (?, ?)")
		for oldKey, newKey := range swapFiles {
			delStmt.Exec(tagName, oldKey)
			insStmt.Exec(tagName, newKey)
		}
		delStmt.Close()
		insStmt.Close()
	}

	if len(swapIgnored) > 0 {
		delStmt, _ := tx.Prepare("DELETE FROM files_ignored WHERE tag_name = ? AND path = ?")
		insStmt, _ := tx.Prepare("INSERT INTO files_ignored (tag_name, path) VALUES (?, ?)")
		for oldKey, newKey := range swapIgnored {
			delStmt.Exec(tagName, oldKey)
			insStmt.Exec(tagName, newKey)
		}
		delStmt.Close()
		insStmt.Close()
	}

	return tx.Commit()
}
