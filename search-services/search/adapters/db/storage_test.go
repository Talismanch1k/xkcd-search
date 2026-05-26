package db

import (
	"database/sql/driver"
	"errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/search/core"
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
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
			},
			want:    []int{1, 2},
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

func TestDB_ListIDs(t *testing.T) {
	t.Parallel()

	const query = `SELECT id, url, title, alt, transcript
			FROM comics
			WHERE id = ANY($1)`

	tests := []struct {
		name    string
		ids     []int
		mock    func(mock sqlmock.Sqlmock)
		want    []core.Comic
		wantErr bool
	}{
		{
			name: "success",
			ids:  []int{1},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id", "url", "title", "alt", "transcript"}).
						AddRow(1, "url1", "{t1}", "{a1}", "{tr1}"))
			},
			want: []core.Comic{
				{ID: 1, URL: "url1", Title: []string{"t1"}, Alt: []string{"a1"}, Transcript: []string{"tr1"}},
			},
			wantErr: false,
		},
		{
			name: "empty input",
			ids:  []int{},
			mock: func(mock sqlmock.Sqlmock) {},
			want: nil,
		},
		{
			name: "error",
			ids:  []int{1},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "scan error",
			ids:  []int{1},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id", "url", "title", "alt", "transcript"}).
						AddRow("not_an_int", "url1", "{t1}", "{a1}", "{tr1}"))
			},
			wantErr: true,
		},
		{
			name: "iteration error",
			ids:  []int{1},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id", "url", "title", "alt", "transcript"}).
						AddRow(1, "url1", "{t1}", "{a1}", "{tr1}").
						RowError(0, errors.New("iteration error")))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := setup(t)
			tt.mock(mock)

			got, err := db.ListIDs(t.Context(), tt.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.ListIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DB.ListIDs() = %v, want %v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestDB_Search(t *testing.T) {
	t.Parallel()

	const query = `SELECT id, url, title, alt, transcript
			FROM comics
			WHERE title				&& $1::text[]
				OR	alt					&& $1::text[]
				OR	transcript	&& $1::text[]`

	tests := []struct {
		name    string
		words   []string
		mock    func(mock sqlmock.Sqlmock)
		want    []core.Comic
		wantErr bool
	}{
		{
			name:  "success",
			words: []string{"word"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id", "url", "title", "alt", "transcript"}).
						AddRow(1, "url1", "{t1}", "{a1}", "{tr1}"))
			},
			want: []core.Comic{
				{ID: 1, URL: "url1", Title: []string{"t1"}, Alt: []string{"a1"}, Transcript: []string{"tr1"}},
			},
			wantErr: false,
		},
		{
			name:  "empty words",
			words: []string{},
			mock:  func(mock sqlmock.Sqlmock) {},
			want:  nil,
		},
		{
			name:  "error",
			words: []string{"word"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:  "scan error",
			words: []string{"word"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id", "url", "title", "alt", "transcript"}).
						AddRow("not_an_int", "url1", "{t1}", "{a1}", "{tr1}"))
			},
			wantErr: true,
		},
		{
			name:  "iteration error",
			words: []string{"word"},
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(query)).
					WithArgs(sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id", "url", "title", "alt", "transcript"}).
						AddRow(1, "url1", "{t1}", "{a1}", "{tr1}").
						RowError(0, errors.New("iteration error")))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := setup(t)
			tt.mock(mock)

			got, err := db.Search(t.Context(), tt.words)
			if (err != nil) != tt.wantErr {
				t.Errorf("DB.Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DB.Search() = %v, want %v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
