// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import "database/sql"

// Database define a interface para operações de banco de dados, permitindo mock em testes.
type Database interface {
	// GetDB retorna a instância do banco de dados
	GetDB() (*sql.DB, error)
	// CloseDB encerra a conexão com o banco de dados
	CloseDB() error
}

// TagRepository define a interface para operações CRUD de tags.
type TagRepository interface {
	// CreateTags cadastra múltiplas tags no banco de forma atômica
	CreateTags(tags []string, meta TagMeta) error
	// DeleteTags remove as tags e aciona em cascata a limpeza de seus arquivos rastreados
	DeleteTags(tags []string) error
	// GetAllTags retorna uma lista com os nomes de todas as tags cadastradas
	GetAllTags() ([]string, error)
	// GetAllTagsWithMeta recupera o dicionário completo de tags e seus metadados
	GetAllTagsWithMeta() (map[string]TagMeta, error)
	// GetTagMeta recupera os metadados de uma tag
	GetTagMeta(tagName string) (TagMeta, error)
	// RenameTag transfere atomicamente os índices de uma tag para um novo nome
	RenameTag(oldName, newName string) error
}

// FileTrackingRepository define a interface para operações de rastreamento de arquivos.
type FileTrackingRepository interface {
	// TrackPaths recebe caminhos já processados pelo resolver e insere no rastreamento
	TrackPaths(tagName string, targets []string) error
	// UntrackPath remove um caminho pré-processado da tag
	UntrackPath(tagName, targetPath string) error
	// GetFilesByTag retorna a lista plana de todos os caminhos rastreados por uma tag
	GetFilesByTag(tagName string) ([]string, error)
	// GetTagRawKeys retorna as chaves do banco exatamente como estão gravadas
	GetTagRawKeys(tagName string) (files []string, ignored []string, err error)
}

// IgnoreRepository define a interface para operações de ignore/denylist.
type IgnoreRepository interface {
	// IgnorePaths move alvos pré-processados para a denylist
	IgnorePaths(tagName string, targets []string) error
	// GetIgnoredPaths carrega o Exclusion Index da tag em um mapa O(1)
	GetIgnoredPaths(tagName string) (map[string]bool, error)
	// UnignorePaths remove alvos pré-processados da denylist
	UnignorePaths(tagName string, targets []string) error
	// RemoveKeysFromTag deleta chaves específicas dos buckets de uma tag
	RemoveKeysFromTag(tagName string, filesToRemove, ignoredToRemove []string) error
}

// GitIgnoreRepository define a interface para operações de ignore específicas do Git.
type GitIgnoreRepository interface {
	// GitIgnorePaths adiciona os caminhos na denylist persistente do repositório
	GitIgnorePaths(repoID string, targets []string) error
	// GetGitIgnoredPaths retorna um hash map contendo a denylist do repositório
	GetGitIgnoredPaths(repoID string) (map[string]bool, error)
	// UnignoreGitPaths remove caminhos da denylist do repositório
	UnignoreGitPaths(repoID string, targets []string) error
}

// UpdateRepository define a interface para operações de atualização em lote.
type UpdateRepository interface {
	// UpdateTagScope reescreve os caminhos de uma tag resolvendo a troca de contexto
	UpdateTagScope(tagName string, meta TagMeta, swapFiles, swapIgnored map[string]string) error
}

// BackupRepository define a interface para operações de backup/restore.
type BackupRepository interface {
	// DumpGitRepositoryData extrai do banco todos os dados atrelados a um repositório Git
	DumpGitRepositoryData(repoID string) (BackupSchema, error)
	// RestoreGitRepositoryData limpa e injeta atomicamente os dados do backup de volta no banco
	RestoreGitRepositoryData(currentGitRoot string, backup BackupSchema) error
}

// Repository agrupa todas as interfaces de repositório para facilitar a injeção de dependências.
type Repository interface {
	Database
	TagRepository
	FileTrackingRepository
	IgnoreRepository
	GitIgnoreRepository
	UpdateRepository
	BackupRepository
}
