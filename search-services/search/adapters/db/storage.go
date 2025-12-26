package db

import (
	"context"
	"fmt"
	"log/slog"
	"search-service/search/core"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	getComicsByIds   = `SELECT id, url FROM comics WHERE id = ANY($1)`
	getAllComicsInfo = `SELECT id, url, words FROM comics`
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

func (db *DB) GetComicsByIds(ctx context.Context, ids []int64) ([]core.Comic, error) {
	var comics []core.Comic
	if err := db.conn.Select(&comics, getComicsByIds, pq.Array(ids)); err != nil {
		return nil, fmt.Errorf("failed to select comics by ids from comics table: %w", err)
	}
	return comics, nil
}

func (db *DB) GetAllComicsInfo(ctx context.Context) ([]core.ComicInfo, error) {
	var comicsPg []struct {
		core.Comic
		Words pq.StringArray `db:"words"`
	}
	if err := db.conn.Select(&comicsPg, getAllComicsInfo); err != nil {
		return nil, fmt.Errorf("failed to select all comic info from comics table: %w", err)
	}

	comics := make([]core.ComicInfo, len(comicsPg))
	for i, info := range comicsPg {
		comics[i] = core.ComicInfo{
			Comic: info.Comic,
			Words: info.Words,
		}
	}
	return comics, nil
}
