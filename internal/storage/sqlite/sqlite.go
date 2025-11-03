package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"url-shortener/internal/storage"

	_ "modernc.org/sqlite"
)

var ErrURLExists = errors.New("url alias already exists")

type Storage struct {
	db *sql.DB
}

func cleanFSPath(p string) string {
	// убираем file: схему и query-параметры, чтобы MkdirAll работал по ФС, а не по URL
	if strings.HasPrefix(p, "file:") {
		if u, err := url.Parse(p); err == nil && u.Scheme == "file" {
			// у file:opaque может быть путь в Opaque
			if u.Path != "" {
				p = u.Path
			} else {
				p = u.Opaque
			}
		} else {
			p = strings.TrimPrefix(p, "file:")
		}
	}
	// отрезаем query, если вдруг есть
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	// нормализуем слэши под текущую ОС
	return filepath.FromSlash(p)
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	// нормализуем путь ТОЛЬКО для создания директорий
	// (DSN в sql.Open оставляем исходный — он может содержать опции драйвера)
	fsPath := cleanFSPath(storagePath)

	if err := os.MkdirAll(filepath.Dir(fsPath), 0o755); err != nil {
		return nil, fmt.Errorf("%s: mkdir %w", op, err)
	}

	db, err := sql.Open("sqlite", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: ping %w", op, err)
	}

	stmt := `
	CREATE TABLE IF NOT EXISTS url(
		id INTEGER PRIMARY KEY, 
		alias TEXT NOT NULL UNIQUE, 
		url TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
	`

	if _, err = db.Exec(stmt); err != nil {
		return nil, fmt.Errorf("%s: create schema:%w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	stmt, err := s.db.Prepare(`
		INSERT INTO url(url, alias)
		VALUES(?, ?)
		ON CONFLICT(alias) DO NOTHING
	`)
	if err != nil {
		return 0, fmt.Errorf("%s: prepare %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(urlToSave, alias)
	if err != nil {
		return 0, fmt.Errorf("%s: exec %w", op, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%s:  rows affected: %w", op, err)
	}
	if affected == 0 {
		return 0, fmt.Errorf("%s: %w", op, ErrURLExists)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s:  last insert id: %w", op, err)
	}
	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.Prepare(`
		SELECT url
		FROM url
		WHERE alias = ?
	`)
	if err != nil {
		return "", fmt.Errorf("%s: prepare %w", op, err)
	}
	defer stmt.Close()

	var resURL string
	err = stmt.QueryRow(alias).Scan(&resURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", storage.ErrURLNotFound
	}
	if err != nil {
		return "", fmt.Errorf("%s: scan %w", op, err)
	}

	return resURL, nil
}

func (s *Storage) DeleteURL(alias string) error {
	const op = "storage.sqlite.DeleteURL"

	stmt, err := s.db.Prepare(`
		DELETE FROM url
		WHERE alias = ?
	`)
	if err != nil {
		return fmt.Errorf("%s: prepare %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(alias)
	if err != nil {
		return fmt.Errorf("%s: exec %w", op, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s:  rows affected: %w", op, err)
	}
	if affected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
	}

	return nil
}

func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
