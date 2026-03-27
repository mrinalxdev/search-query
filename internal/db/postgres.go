package database

import (
	"database/sql"
	"fmt"
	"mulit-minIO/internal/config"
	"time"

	_ "github.com/lib/pq"
)

type Database struct {
	*sql.DB
}

type FileRecord struct {
	ID          int64     `json:"id"`
	ObjectName  string    `json:"object_name"`
	OriginalName string   `json:"original_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	ETag        string    `json:"etag"`
	UploadedBy  string    `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewDatabase(cfg *config.DBConfig) (*Database, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &Database{DB: db}, nil
}

func (d *Database) Migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS files (
		id BIGSERIAL PRIMARY KEY,
		object_name VARCHAR(500) NOT NULL UNIQUE,
		original_name VARCHAR(500) NOT NULL,
		content_type VARCHAR(255) NOT NULL,
		size BIGINT NOT NULL,
		etag VARCHAR(255) NOT NULL,
		uploaded_by VARCHAR(255) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_files_object_name ON files(object_name);
	CREATE INDEX IF NOT EXISTS idx_files_uploaded_by ON files(uploaded_by);
	CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at);
	`

	_, err := d.Exec(query)
	return err
}

func (d *Database) CreateFileRecord(file *FileRecord) error {
	query := `
	INSERT INTO files (object_name, original_name, content_type, size, etag, uploaded_by)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_at, updated_at
	`

	err := d.QueryRow(query, 
		file.ObjectName,
		file.OriginalName,
		file.ContentType,
		file.Size,
		file.ETag,
		file.UploadedBy,
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)

	return err
}

func (d *Database) GetFileRecord(objectName string) (*FileRecord, error) {
	query := `
	SELECT id, object_name, original_name, content_type, size, etag, uploaded_by, created_at, updated_at
	FROM files
	WHERE object_name = $1
	`

	file := &FileRecord{}
	err := d.QueryRow(query, objectName).Scan(
		&file.ID,
		&file.ObjectName,
		&file.OriginalName,
		&file.ContentType,
		&file.Size,
		&file.ETag,
		&file.UploadedBy,
		&file.CreatedAt,
		&file.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return file, nil
}

func (d *Database) DeleteFileRecord(objectName string) error {
	query := `DELETE FROM files WHERE object_name = $1`
	_, err := d.Exec(query, objectName)
	return err
}

func (d *Database) ListFileRecords(limit, offset int) ([]FileRecord, error) {
	query := `
	SELECT id, object_name, original_name, content_type, size, etag, uploaded_by, created_at, updated_at
	FROM files
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2
	`

	rows, err := d.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var file FileRecord
		err := rows.Scan(
			&file.ID,
			&file.ObjectName,
			&file.OriginalName,
			&file.ContentType,
			&file.Size,
			&file.ETag,
			&file.UploadedBy,
			&file.CreatedAt,
			&file.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, rows.Err()
}