// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
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

	// Puxa Denylist do Repo
	rows, err := db.Query("SELECT path FROM git_ignored WHERE repo_id = ?", repoID)
	if err == nil {
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err == nil {
				backup.RepoDenylist = append(backup.RepoDenylist, p)
			}
		}
		rows.Close()
	}

	// Puxa Tags do Repo
	tagRows, err := db.Query("SELECT name, type, repo_name, git_root FROM tags WHERE type = ? AND repo_id = ?", TagTypeGit, repoID)
	if err != nil {
		return backup, err
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var tagName string
		var meta TagMeta
		var repoName, gitRoot sql.NullString
		meta.RepoID = repoID

		if err := tagRows.Scan(&tagName, &meta.Type, &repoName, &gitRoot); err != nil {
			continue
		}

		if repoName.Valid {
			meta.RepoName = repoName.String
			backup.RepoName = repoName.String
		}
		if gitRoot.Valid {
			meta.GitRoot = gitRoot.String
		}

		tb := TagBackup{Meta: meta}

		// Puxa Arquivos Rastreados
		fRows, _ := db.Query("SELECT path FROM files_tracked WHERE tag_name = ?", tagName)
		for fRows != nil && fRows.Next() {
			var p string
			fRows.Scan(&p)
			tb.Files = append(tb.Files, p)
		}
		if fRows != nil {
			fRows.Close()
		}

		// Puxa Arquivos Ignorados
		iRows, _ := db.Query("SELECT path FROM files_ignored WHERE tag_name = ?", tagName)
		for iRows != nil && iRows.Next() {
			var p string
			iRows.Scan(&p)
			tb.Ignored = append(tb.Ignored, p)
		}
		if iRows != nil {
			iRows.Close()
		}

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

	// Restaura Denylist
	if len(backup.RepoDenylist) > 0 {
		stmt, _ := tx.Prepare("INSERT OR IGNORE INTO git_ignored (repo_id, path) VALUES (?, ?)")
		for _, p := range backup.RepoDenylist {
			stmt.Exec(backup.RepoID, p)
		}
		stmt.Close()
	}

	// Restaura Tags
	if len(backup.Tags) > 0 {
		tagStmt, _ := tx.Prepare("INSERT OR REPLACE INTO tags (name, type, repo_id, repo_name, git_root) VALUES (?, ?, ?, ?, ?)")
		fStmt, _ := tx.Prepare("INSERT OR IGNORE INTO files_tracked (tag_name, path) VALUES (?, ?)")
		iStmt, _ := tx.Prepare("INSERT OR IGNORE INTO files_ignored (tag_name, path) VALUES (?, ?)")

		for tagName, tagData := range backup.Tags {
			meta := tagData.Meta
			tagStmt.Exec(tagName, meta.Type, meta.RepoID, meta.RepoName, currentGitRoot)

			for _, p := range tagData.Files {
				fStmt.Exec(tagName, p)
			}
			for _, p := range tagData.Ignored {
				iStmt.Exec(tagName, p)
			}
		}
		tagStmt.Close()
		fStmt.Close()
		iStmt.Close()
	}

	return tx.Commit()
}
