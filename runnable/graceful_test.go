package runnable

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraceful(t *testing.T) {
	var startCount, stopCount int

	// Simple start function.
	start := func(context.Context) error {
		startCount++
		return nil
	}

	// Simple stop function.
	stop := func() error {
		stopCount++
		return nil
	}

	// WaitGroup to wait for the stop to complete.
	var wg sync.WaitGroup

	wg.Add(1)
	gr := NewGraceful(start, stop, false, &wg, nil)

	// Create a context to stop the component with.
	ctx, cancelFunc := context.WithCancel(context.TODO())
	assert.Nil(t, gr.Start(ctx))

	// Cancel the context and stop the component.
	cancelFunc()
	wg.Wait()

	assert.Equal(t, 1, startCount)
	assert.Equal(t, 1, stopCount)
}
