package db

import (
	"context"
	"database/sql"
	"fmt"
	"log" // Добавляем импорт log
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Manager struct {
	db *sql.DB
}

func NewManager() (*Manager, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Manager{db: db}, nil
}

func (m *Manager) Close() error {
	return m.db.Close()
}

func (m *Manager) InitSchema(ctx context.Context) error {
	// Пробуем несколько путей к файлу схемы
	possiblePaths := []string{
		"/app/internal/db/schema.sql",      // Docker контейнер (основной путь)
		"internal/db/schema.sql",           // локальная разработка
		"./schema.sql",                     // альтернативный
	}

	var schemaSQL []byte
	var err error

	for _, path := range possiblePaths {
		schemaSQL, err = os.ReadFile(path)
		if err == nil {
			log.Printf("Schema loaded from: %s", path)
			break
		}
	}

	if err != nil {
		return fmt.Errorf("failed to read schema file. Tried paths: %v. Error: %w", possiblePaths, err)
	}

	_, err = m.db.ExecContext(ctx, string(schemaSQL))
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

func (m *Manager) GetDB() *sql.DB {
	return m.db
}