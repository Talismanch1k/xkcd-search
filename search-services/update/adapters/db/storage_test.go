package db

import (
	"database/sql/driver"
	"errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
)

type anyValue struct{}

func (anyValue) ConvertValue(v any) (driver.Value, error) {
	return v, nil
}

func setup(t *testing.T) (*DB, sqlmock.Sqlmock) {
	t.Helper()

	mockDB, mock, err := sqlmock.New(sqlmock.ValueConverterOption(anyValue{}))
	if err != nil {
		t.Fatalf("failed to open mock database: %v", err)
	}

	db := &DB{
		conn: sqlx.NewDb(mockDB, "postgres"),
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock
}

func TestDB_Add(t *testing.T) {
	t.Parallel()

	const query = `INSERT INTO comics (id, url, title, alt, transcript)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO NOTHING`

	tests := []struct {
		name    string
		comics  core.Comics
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success",
			comics: core.Comics{
				ID:         1,
				URL:        "https://xkcd.com/1/",
				Title:      []string{"title"},
				Alt:        []string{"alt"},
				Transcript: []string{"transcript"},
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(query)).
					WithArgs(1, "https://xkcd.com/1/", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "db error",
			comics: core.Comics{
				ID: 1,
			},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(query)).
					WithArgs(1, "", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := setup(t)
			tt.mock(mock)

			err := db.Add(t.Context(), tt.comics)
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.Add() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestDB_Stats(t *testing.T) {
	t.Parallel()

	const query1 = `SELECT COUNT(*), 
		COALESCE(SUM(cardinality(title)), 0) +
		COALESCE(SUM(cardinality(transcript)), 0) + 
		COALESCE(SUM(cardinality(alt)), 0)
		FROM comics`
	const query2 = `SELECT COUNT(DISTINCT word) FROM comics, unnest(title || transcript || alt) AS word`

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		want    core.DBStats
		wantErr bool
	}{
		{
			name: "success",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query1)).
					WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(10, 100))
				mock.ExpectQuery(regexp.QuoteMeta(query2)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))
			},
			want: core.DBStats{
				ComicsFetched: 10,
				WordsTotal:    100,
				WordsUnique:   50,
			},
			wantErr: false,
		},
		{
			name: "zero comics",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query1)).
					WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(0, 0))
			},
			want: core.DBStats{
				ComicsFetched: 0,
				WordsTotal:    0,
				WordsUnique:   0,
			},
			wantErr: false,
		},
		{
			name: "query 1 error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query1)).
					WillReturnError(errors.New("db error"))
			},
			want:    core.DBStats{},
			wantErr: true,
		},
		{
			name: "query 2 error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query1)).
					WillReturnRows(sqlmock.NewRows([]string{"count", "sum"}).AddRow(10, 100))
				mock.ExpectQuery(regexp.QuoteMeta(query2)).
					WillReturnError(errors.New("db error"))
			},
			want:    core.DBStats{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := setup(t)
			tt.mock(mock)

			got, err := db.Stats(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.Stats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("DB.Stats() = %v, want %v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestDB_IDs(t *testing.T) {
	t.Parallel()

	const query = `SELECT id FROM comics`

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		want    []int
		wantErr bool
	}{
		{
			name: "success",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2).AddRow(3))
			},
			want:    []int{1, 2, 3},
			wantErr: false,
		},
		{
			name: "empty",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
			},
			want:    []int(nil),
			wantErr: false,
		},
		{
			name: "error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnError(errors.New("db error"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := setup(t)
			tt.mock(mock)

			got, err := db.IDs(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.IDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DB.IDs() = %v, want %v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestDB_Drop(t *testing.T) {
	t.Parallel()

	const query = `TRUNCATE TABLE comics`

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(query)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(query)).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := setup(t)
			tt.mock(mock)

			err := db.Drop(t.Context())
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.Drop() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
