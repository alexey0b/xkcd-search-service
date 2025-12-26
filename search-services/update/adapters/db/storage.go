package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"search-service/update/core"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

const (
	// insert
	insertComic = `
		INSERT INTO comics (id, url, words) 
		VALUES (:id, :url, :words)
	`

	// select
	getIDs         = `SELECT id FROM comics`
	getComicsStats = `SELECT * FROM comics_stats`

	// update
	updateStats = `
		WITH stats AS (
			SELECT 
			COUNT(*) as comics_fetched,
			COALESCE(SUM(array_length(words, 1)), 0) as words_total,
			(
				SELECT COUNT(DISTINCT word) 
				FROM (SELECT unnest(words) as word FROM comics) t
			) as words_unique
			FROM comics
		)

		UPDATE comics_stats
		SET 
		comics_fetched = stats.comics_fetched,
		words_total = stats.words_total,
		words_unique = stats.words_unique
		FROM stats
	`
	resetComicsStats = `
        UPDATE comics_stats 
        SET 
        comics_fetched = 0,
        words_total = 0,
        words_unique = 0
    `

	// truncate
	truncateComics = `TRUNCATE comics`
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}
	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Close() {
	if err := db.conn.Close(); err != nil {
		db.log.Warn("failed to close database connection", "error", err)
	}
}

func (db *DB) Add(ctx context.Context, comic ...core.Comic) error {
	tx, err := db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.log.Error("failed to rollback transaction", "error", err)
		}
	}()

	if _, err = tx.NamedExecContext(ctx, insertComic, comic); err != nil {
		return fmt.Errorf("failed to insert into comic table : %w", err)
	}
	if _, err = tx.ExecContext(ctx, updateStats); err != nil {
		return fmt.Errorf("failed to update comics_stats table: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var stats core.DBStats
	err := db.conn.GetContext(ctx, &stats, getComicsStats)
	if err != nil {
		return core.DBStats{}, fmt.Errorf("failed to select stats from comics_stats table: %w", err)
	}
	return stats, nil
}

func (db *DB) IDs(ctx context.Context) ([]int64, error) {
	var IDs []int64
	err := db.conn.SelectContext(ctx, &IDs, getIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to select IDs from comics table: %w", err)
	}
	return IDs, nil
}

func (db *DB) Drop(ctx context.Context) error {
	tx, err := db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			db.log.Error("failed to rollback transaction", "error", err)
		}
	}()

	_, err = tx.ExecContext(ctx, truncateComics)
	if err != nil {
		return fmt.Errorf("failed to truncate comics table: %w", err)
	}
	_, err = tx.ExecContext(ctx, resetComicsStats)
	if err != nil {
		return fmt.Errorf("failed to truncate comics_stats table: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
