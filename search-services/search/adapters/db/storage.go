package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/pkg/closer"
	"yadro.com/course/search/core"
)

type DB struct {
	conn *sqlx.DB
}

func New(address string) (*DB, error) {
	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		return nil, fmt.Errorf("connect to db at: %w", err)
	}

	db.SetConnMaxLifetime(5 * time.Minute)

	return &DB{conn: db}, nil
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	var ids []int

	err := db.conn.SelectContext(ctx, &ids, `SELECT id FROM comics`)
	if err != nil {
		return nil, fmt.Errorf("select comic ids: %w", err)
	}

	return ids, nil
}

func (db *DB) ListIDs(ctx context.Context, ids []int) ([]core.Comic, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
			SELECT id, url, title, alt, transcript
			FROM comics
			WHERE id = ANY($1)
	`
	rows, err := db.conn.QueryContext(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("query comics: %w", err)
	}
	defer closer.CloseOrLog(rows)

	m := pgtype.NewMap()
	var comics []core.Comic
	for rows.Next() {
		var c core.Comic

		err := rows.Scan(
			&c.ID,
			&c.URL,
			m.SQLScanner(&c.Title),
			m.SQLScanner(&c.Alt),
			m.SQLScanner(&c.Transcript),
		)
		if err != nil {
			return nil, fmt.Errorf("scan commic: %w", err)
		}

		comics = append(comics, c)
	}

	// fail during interation
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows interation: %w", err)
	}

	return comics, nil
}

func (db *DB) Search(ctx context.Context, words []string) ([]core.Comic, error) {
	if len(words) == 0 {
		return nil, nil
	}

	query := `
			SELECT id, url, title, alt, transcript
			FROM comics
			WHERE title				&& $1::text[]
				OR	alt					&& $1::text[]
				OR	transcript	&& $1::text[]
	`
	rows, err := db.conn.QueryContext(ctx, query, words)
	if err != nil {
		return nil, fmt.Errorf("query comics: %w", err)
	}
	defer closer.CloseOrLog(rows)

	m := pgtype.NewMap()
	var comics []core.Comic
	for rows.Next() {
		var c core.Comic

		err := rows.Scan(
			&c.ID,
			&c.URL,
			m.SQLScanner(&c.Title),
			m.SQLScanner(&c.Alt),
			m.SQLScanner(&c.Transcript),
		)
		if err != nil {
			return nil, fmt.Errorf("scan commic: %w", err)
		}

		comics = append(comics, c)
	}

	// fail during interation
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows interation: %w", err)
	}

	return comics, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
