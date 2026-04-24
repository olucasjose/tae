// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
)

// BackupSchema define a estrutura do JSON exportado.
type BackupSchema struct {
	RepoID       string               `json:"repo_id"`
	RepoName     string               `json:"repo_name,omitempty"`
	RepoDenylist []string             `json:"repo_denylist,omitempty"`
	Tags         map[string]TagBackup `json:"tags,omitempty"`
}

type TagBackup struct {
	Meta    TagMeta  `json:"meta"`
	Files   []string `json:"files,omitempty"`
	Ignored []string `json:"ignored,omitempty"`
}

// DumpGitRepositoryData extrai do banco todos os dados atrelados a um repositório Git específico.
func DumpGitRepositoryData(repoID string) (BackupSchema, error) {
	db, err := GetDB()
	if err != nil {
		return BackupSchema{}, err
	}

	backup := BackupSchema{
		RepoID: repoID,
		Tags:   make(map[string]TagBackup),
	}

	rows, err := db.Query("SELECT path FROM git_ignored WHERE repo_id = ?", repoID)
	if err != nil {
		return backup, fmt.Errorf("erro ao consultar denylist do repo: %w", err)
	}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return backup, fmt.Errorf("erro ao escanear caminho da denylist do repo: %w", err)
		}
		backup.RepoDenylist = append(backup.RepoDenylist, p)
	}
	rows.Close()

	tagRows, err := db.Query("SELECT name, type, repo_name, git_root FROM tags WHERE type = ? AND repo_id = ?", TagTypeGit, repoID)
	if err != nil {
		return backup, fmt.Errorf("erro ao consultar tags do repo: %w", err)
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var tagName string
		var meta TagMeta
		var repoName, gitRoot sql.NullString
		meta.RepoID = repoID

		if err := tagRows.Scan(&tagName, &meta.Type, &repoName, &gitRoot); err != nil {
			return backup, fmt.Errorf("erro ao escanear metadados da tag: %w", err)
		}

		if repoName.Valid {
			meta.RepoName = repoName.String
			backup.RepoName = repoName.String
		}
		if gitRoot.Valid {
			meta.GitRoot = gitRoot.String
		}

		tb := TagBackup{Meta: meta}

		fRows, err := db.Query("SELECT path FROM files_tracked WHERE tag_name = ?", tagName)
		if err != nil {
			return backup, fmt.Errorf("erro ao consultar arquivos rastreados da tag '%s': %w", tagName, err)
		}
		for fRows.Next() {
			var p string
			if err := fRows.Scan(&p); err != nil {
				fRows.Close()
				return backup, fmt.Errorf("erro ao escanear arquivo rastreado da tag '%s': %w", tagName, err)
			}
			tb.Files = append(tb.Files, p)
		}
		fRows.Close()

		iRows, err := db.Query("SELECT path FROM files_ignored WHERE tag_name = ?", tagName)
		if err != nil {
			return backup, fmt.Errorf("erro ao consultar arquivos ignorados da tag '%s': %w", tagName, err)
		}
		for iRows.Next() {
			var p string
			if err := iRows.Scan(&p); err != nil {
				iRows.Close()
				return backup, fmt.Errorf("erro ao escanear arquivo ignorado da tag '%s': %w", tagName, err)
			}
			tb.Ignored = append(tb.Ignored, p)
		}
		iRows.Close()

		backup.Tags[tagName] = tb
	}

	return backup, nil
}

// RestoreGitRepositoryData limpa e injeta atomicamente os dados do backup de volta no banco.
func RestoreGitRepositoryData(currentGitRoot string, backup BackupSchema) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if len(backup.RepoDenylist) > 0 {
		stmt, err := tx.Prepare("INSERT OR IGNORE INTO git_ignored (repo_id, path) VALUES (?, ?)")
		if err != nil {
			return fmt.Errorf("erro ao preparar statement de inserção na denylist: %w", err)
		}
		for _, p := range backup.RepoDenylist {
			if _, err := stmt.Exec(backup.RepoID, p); err != nil {
				stmt.Close()
				return fmt.Errorf("erro ao inserir caminho '%s' na denylist: %w", p, err)
			}
		}
		stmt.Close()
	}

	if len(backup.Tags) > 0 {
		tagStmt, err := tx.Prepare("INSERT OR REPLACE INTO tags (name, type, repo_id, repo_name, git_root) VALUES (?, ?, ?, ?, ?)")
		if err != nil {
			return fmt.Errorf("erro ao preparar statement de tags: %w", err)
		}
		fStmt, err := tx.Prepare("INSERT OR IGNORE INTO files_tracked (tag_name, path) VALUES (?, ?)")
		if err != nil {
			tagStmt.Close()
			return fmt.Errorf("erro ao preparar statement de files_tracked: %w", err)
		}
		iStmt, err := tx.Prepare("INSERT OR IGNORE INTO files_ignored (tag_name, path) VALUES (?, ?)")
		if err != nil {
			tagStmt.Close()
			fStmt.Close()
			return fmt.Errorf("erro ao preparar statement de files_ignored: %w", err)
		}

		for tagName, tagData := range backup.Tags {
			meta := tagData.Meta
			if _, err := tagStmt.Exec(tagName, meta.Type, meta.RepoID, meta.RepoName, currentGitRoot); err != nil {
				tagStmt.Close()
				fStmt.Close()
				iStmt.Close()
				return fmt.Errorf("erro ao inserir a tag '%s': %w", tagName, err)
			}

			for _, p := range tagData.Files {
				if _, err := fStmt.Exec(tagName, p); err != nil {
					tagStmt.Close()
					fStmt.Close()
					iStmt.Close()
					return fmt.Errorf("erro ao rastrear arquivo '%s' para a tag '%s': %w", p, tagName, err)
				}
			}
			for _, p := range tagData.Ignored {
				if _, err := iStmt.Exec(tagName, p); err != nil {
					tagStmt.Close()
					fStmt.Close()
					iStmt.Close()
					return fmt.Errorf("erro ao ignorar arquivo '%s' para a tag '%s': %w", p, tagName, err)
				}
			}
		}
		tagStmt.Close()
		fStmt.Close()
		iStmt.Close()
	}

	return tx.Commit()
}
