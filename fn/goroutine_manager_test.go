package fn

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGoroutineManager tests that GoroutineManager starts goroutines, until ctx
// expires fails to start after it expires and waits for already started
// goroutines in Stop method.
func TestGoroutineManager(t *testing.T) {
	t.Parallel()

	m := NewGoroutineManager(context.Background())

	taskChan := make(chan struct{})

	require.NoError(t, m.Go(func(ctx context.Context) {
		<-taskChan
	}))

	t1 := time.Now()

	// Close taskChan in 1s, causing the goroutine to stop.
	time.AfterFunc(time.Second, func() {
		close(taskChan)
	})

	m.Stop()
	stopDelay := time.Since(t1)

	// Make sure Stop was waiting for the goroutine to stop.
	require.Greater(t, stopDelay, time.Second)

	// Make sure new goroutines do not start after Stop.
	require.ErrorIs(t, m.Go(func(ctx context.Context) {}), ErrStopping)
}

// TestGoroutineManagerContextExpires tests the effect of context expiry.
func TestGoroutineManagerContextExpires(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	m := NewGoroutineManager(ctx)

	require.NoError(t, m.Go(func(ctx context.Context) {
		<-ctx.Done()
	}))

	// The context of the manager should not expire, so the following call
	// must block.
	select {
	case <-m.Context().Done():
		t.Errorf("context must not expire at this point")
	default:
	}

	cancel()

	// The context of the manager should expire, so the following call
	// must not block.
	select {
	case <-m.Context().Done():
	default:
		t.Errorf("context must expire at this point")
	}

	// Make sure new goroutines do not start after context expiry.
	require.ErrorIs(t, m.Go(func(ctx context.Context) {}), ErrStopping)

	// Stop will wait for all goroutines to stop.
	m.Stop()
}

// TestGoroutineManagerStress starts many goroutines while calling Stop.
func TestGoroutineManagerStress(t *testing.T) {
	t.Parallel()

	m := NewGoroutineManager(context.Background())

	stopChan := make(chan struct{})

	time.AfterFunc(1*time.Millisecond, func() {
		m.Stop()
		close(stopChan)
	})

	// Starts 100 goroutines sequentially. Sequential order is needed to
	// keep wg.counter low (0 or 1) to increase probability of race
	// condition to be caught if it exists. If mutex is removed in the
	// implementation, this test crashes under `-race`.
	for i := 0; i < 100; i++ {
		taskChan := make(chan struct{})
		err := m.Go(func(ctx context.Context) {
			close(taskChan)
		})
		// If goroutine was started, wait for its completion.
		if err == nil {
			<-taskChan
		}
	}

	// Wait for Stop to complete.
	<-stopChan
}
