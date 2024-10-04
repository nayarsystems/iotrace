package nagle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeout(t *testing.T) {
	t0 := time.Now()
	timeout := time.Millisecond * 500
	end := make(chan struct{})
	n := NewNagle(10, timeout, func(b []byte) {
		require.Equal(t, "1234", string(b))
		require.GreaterOrEqual(t, time.Since(t0).Milliseconds(), int64(timeout.Milliseconds()))
		close(end)
	})
	n.Write([]byte("1234"))
	select {
	case <-end:
	case <-time.After(timeout * 2):
		t.Fatalf("data no flushed")
	}
}

func TestBufferSize(t *testing.T) {
	t0 := time.Now()
	timeout := time.Millisecond * 500
	end := make(chan struct{})
	n := NewNagle(10, timeout, func(b []byte) {
		require.Equal(t, "1234567890", string(b))
		require.Less(t, time.Since(t0).Milliseconds(), int64(timeout.Milliseconds()))
		close(end)
	})
	n.Write([]byte("1234567890"))
	select {
	case <-end:
	case <-time.After(timeout * 2):
		t.Fatalf("data no flushed")
	}
}
