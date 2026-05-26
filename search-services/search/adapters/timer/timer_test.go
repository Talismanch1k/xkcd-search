package timer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type mockIndexer struct {
	mu         sync.Mutex
	buildCalls int
	buildErr   error
	callChan   chan struct{}
}

func (m *mockIndexer) BuildIndex(ctx context.Context) error {
	m.mu.Lock()
	m.buildCalls++
	m.mu.Unlock()
	if m.callChan != nil {
		m.callChan <- struct{}{}
	}
	return m.buildErr
}

func TestInitiator_Start(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		interval      time.Duration
		mockErr       error
		callsToWait   int
		expectedCalls int
	}{
		{
			name:          "success_multiple_ticks",
			interval:      10 * time.Millisecond,
			mockErr:       nil,
			callsToWait:   3,
			expectedCalls: 3,
		},
		{
			name:          "handles_initial_error",
			interval:      100 * time.Millisecond,
			mockErr:       errors.New("initial error"),
			callsToWait:   1,
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			callChan := make(chan struct{}, 10)
			mock := &mockIndexer{
				callChan: callChan,
				buildErr: tt.mockErr,
			}
			initiator := New(mock, tt.interval)

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			initiator.Start(ctx)

			for i := 0; i < tt.callsToWait; i++ {
				select {
				case <-callChan:
				case <-time.After(200 * time.Millisecond):
					t.Fatalf("call %d timed out", i+1)
				}
			}

			mock.mu.Lock()
			actualCalls := mock.buildCalls
			mock.mu.Unlock()

			if actualCalls < tt.expectedCalls {
				t.Errorf("expected at least %d calls, got %d", tt.expectedCalls, actualCalls)
			}
		})
	}
}

func TestInitiator_RebuildError(t *testing.T) {
	t.Parallel()
	callChan := make(chan struct{}, 10)
	mock := &mockIndexer{callChan: callChan}
	interval := 10 * time.Millisecond
	initiator := New(mock, interval)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	initiator.Start(ctx)

	<-callChan

	mock.mu.Lock()
	mock.buildErr = errors.New("rebuild error")
	mock.mu.Unlock()

	select {
	case <-callChan:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("rebuild call timed out")
	}
}
