// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
)

const (
	TagTypeLocal = "local"
	TagTypeGit   = "git"
)

// TagMeta define a estrutura lógica dos dados da tag na tabela tags
type TagMeta struct {
	Type     string `json:"type"`
	RepoID   string `json:"repo_id,omitempty"`
	RepoName string `json:"repo_name,omitempty"`
	GitRoot  string `json:"git_root,omitempty"`
}

// GetTagMeta recupera os metadados de uma tag.
func GetTagMeta(tagName string) (TagMeta, error) {
	db, err := GetDB()
	if err != nil {
		return TagMeta{}, err
	}

	var meta TagMeta
	var repoID, repoName, gitRoot sql.NullString

	err = db.QueryRow("SELECT type, repo_id, repo_name, git_root FROM tags WHERE name = ?", tagName).Scan(&meta.Type, &repoID, &repoName, &gitRoot)
	if err == sql.ErrNoRows {
		return TagMeta{Type: TagTypeLocal}, nil
	} else if err != nil {
		return TagMeta{}, err
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

	return meta, nil
}
