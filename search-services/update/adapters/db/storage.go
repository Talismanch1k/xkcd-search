package db

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
)

type DB struct {
	conn *sqlx.DB
}

func New(address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		return nil, fmt.Errorf("connect to db at %s: %w", address, err)
	}

	db.SetConnMaxLifetime(5 * time.Minute)

	return &DB{conn: db}, nil
}

func (db *DB) Add(ctx context.Context, comics core.Comics) error {
	_, err := db.conn.ExecContext(
		ctx,
		`INSERT INTO comics (id, url, title, alt, transcript)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO NOTHING`,
		comics.ID, comics.URL, comics.Title, comics.Alt, comics.Transcript,
	)
	if err != nil {
		return fmt.Errorf("insert comic %d: %w", comics.ID, err)
	}

	return nil
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var stats core.DBStats

	err := db.conn.QueryRowContext(ctx,
		`SELECT COUNT(*), 
		COALESCE(SUM(cardinality(title)), 0) +
		COALESCE(SUM(cardinality(transcript)), 0) + 
    COALESCE(SUM(cardinality(alt)), 0)
		FROM comics`,
	).Scan(&stats.ComicsFetched, &stats.WordsTotal)
	if err != nil {
		return core.DBStats{}, fmt.Errorf("count comics: %w", err)
	}

	if stats.ComicsFetched == 0 {
		return stats, nil
	}

	err = db.conn.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT word) FROM comics, unnest(title || transcript || alt) AS word`,
	).Scan(&stats.WordsUnique)
	if err != nil {
		return core.DBStats{}, fmt.Errorf("count unique words: %w", err)
	}

	return stats, nil
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	var ids []int

	err := db.conn.SelectContext(ctx, &ids, `SELECT id FROM comics`)
	if err != nil {
		return nil, fmt.Errorf("select comic ids: %w", err)
	}

	return ids, nil
}

// Drop DELETE ALL VALUES in table 0_0
func (db *DB) Drop(ctx context.Context) error {
	if _, err := db.conn.ExecContext(ctx, `TRUNCATE TABLE comics`); err != nil {
		return fmt.Errorf("delete comics: %w", err)
	}
	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
