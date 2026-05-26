package closer_test

import (
	"errors"
	"io"
	"testing"

	"yadro.com/course/pkg/closer"
)

type mockCloser struct {
	err    error
	called bool
}

func (m *mockCloser) Close() error {
	m.called = true
	return m.err
}

func TestCloseOrLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *mockCloser
	}{
		{
			name:  "nil closer",
			input: nil,
		},
		{
			name:  "calls close",
			input: &mockCloser{},
		},
		{
			name:  "calls close with error",
			input: &mockCloser{err: errors.New("close error")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var c io.Closer
			if tt.input != nil {
				c = tt.input
			}

			closer.CloseOrLog(c)

			if tt.input != nil && !tt.input.called {
				t.Error("expected Close() to be called")
			}
		})
	}
}

func TestCloseOrIgnore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *mockCloser
	}{
		{
			name:  "nil closer",
			input: nil,
		},
		{
			name:  "calls close",
			input: &mockCloser{},
		},
		{
			name:  "calls close with error",
			input: &mockCloser{err: errors.New("close error")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var c io.Closer
			if tt.input != nil {
				c = tt.input
			}

			closer.CloseOrIgnore(c)

			if tt.input != nil && !tt.input.called {
				t.Error("expected Close() to be called")
			}
		})
	}
}
