// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	dbInstance *sql.DB
	dbOnce     sync.Once
	dbErr      error
)

// getDBPath resolve o caminho seguro para o banco de dados global
func getDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("falha ao localizar diretório home: %w", err)
	}

	dir := filepath.Join(home, ".tae")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório base: %w", err)
	}

	return filepath.Join(dir, "tae.db"), nil
}

// GetDB retorna a instância global e thread-safe do pool de conexões do SQLite.
// Substitui o antigo Open() para evitar o gargalo de I/O de múltiplas aberturas.
func GetDB() (*sql.DB, error) {
	dbOnce.Do(func() {
		var dbPath string
		dbPath, dbErr = getDBPath()
		if dbErr != nil {
			return
		}

		// WAL melhora absurdamente a performance concorrente.
		// busy_timeout impede falhas imediatas em locks.
		dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)

		dbInstance, dbErr = sql.Open("sqlite", dsn)
		if dbErr != nil {
			dbErr = fmt.Errorf("falha ao abrir arquivo sqlite: %w", dbErr)
			return
		}

		// Restringe o pool para evitar contenção de trava no nível de arquivo
		dbInstance.SetMaxOpenConns(1)

		if dbErr = dbInstance.Ping(); dbErr != nil {
			dbInstance.Close()
			dbErr = fmt.Errorf("falha ao conectar no banco sqlite: %w", dbErr)
			return
		}

		if dbErr = createSchema(dbInstance); dbErr != nil {
			dbInstance.Close()
			return
		}
	})

	return dbInstance, dbErr
}

// CloseDB encerra o pool de conexões de forma limpa.
// Deve ser chamado apenas uma vez no encerramento do processo.
func CloseDB() error {
	if dbInstance != nil {
		return dbInstance.Close()
	}
	return nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tags (
		name TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		repo_id TEXT,
		repo_name TEXT,
		git_root TEXT
	);

	CREATE TABLE IF NOT EXISTS files_tracked (
		tag_name TEXT,
		path TEXT,
		PRIMARY KEY (tag_name, path),
		FOREIGN KEY (tag_name) REFERENCES tags(name) ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS files_ignored (
		tag_name TEXT,
		path TEXT,
		PRIMARY KEY (tag_name, path),
		FOREIGN KEY (tag_name) REFERENCES tags(name) ON UPDATE CASCADE ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS git_ignored (
		repo_id TEXT,
		path TEXT,
		PRIMARY KEY (repo_id, path)
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("falha ao criar tabelas internas: %w", err)
	}
	return nil
}

// GetAllTags retorna uma lista com os nomes de todas as tags cadastradas no banco
func GetAllTags() ([]string, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	// NÃO use defer db.Close() aqui em hipótese alguma.

	rows, err := db.Query("SELECT name FROM tags")
	if err != nil {
		return nil, fmt.Errorf("falha ao listar tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}
