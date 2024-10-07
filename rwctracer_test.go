package iotrace

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testRWC struct {
	tx     *bytes.Buffer
	rx     *bytes.Buffer
	closed bool
}

func newTestRWC() *testRWC {
	return &testRWC{
		tx: &bytes.Buffer{},
		rx: &bytes.Buffer{},
	}
}

func (t *testRWC) Write(data []byte) (n int, e error) {
	return t.tx.Write(data)
}

func (t *testRWC) Read(data []byte) (n int, e error) {
	return t.rx.Read(data)
}

func (t *testRWC) Close() error {
	t.closed = true
	return nil
}

func TestFlushFromTimeout(t *testing.T) {
	rwc := newTestRWC()
	flushTimeout := time.Millisecond * 100
	writeCh := make(chan []byte, 1)
	readCh := make(chan []byte, 1)
	rwct := NewRWCTracer(rwc, 10, flushTimeout, func(b []byte) {
		writeCh <- b
	}, func(b []byte) {
		readCh <- b
	})

	// Test write hook
	t0 := time.Now()
	rwct.Write([]byte("1234"))
	select {
	case b := <-writeCh:
		require.Equal(t, "1234", string(b))
		require.GreaterOrEqual(t, time.Since(t0).Milliseconds(), int64(flushTimeout.Milliseconds()))
	case <-time.After(flushTimeout * 2):
		t.Fatalf("data no flushed")
	}
	readData := make([]byte, 100)

	// Test read hook producing some data to read
	rwc.rx.Write([]byte("5678"))
	t0 = time.Now()
	n, err := rwct.Read(readData)
	require.NoError(t, err)
	require.Equal(t, "5678", string(readData[:n]))
	select {
	case b := <-readCh:
		require.Equal(t, "5678", string(b))
		require.GreaterOrEqual(t, time.Since(t0).Milliseconds(), int64(flushTimeout.Milliseconds()))
	case <-time.After(flushTimeout * 2):
		t.Fatalf("data no flushed")
	}
}

func TestFlushFromBufferSize(t *testing.T) {
	rwc := newTestRWC()
	flushTimeout := time.Millisecond * 100
	writeCh := make(chan []byte, 1)
	readCh := make(chan []byte, 1)
	rwct := NewRWCTracer(rwc, 10, flushTimeout, func(b []byte) {
		writeCh <- b
	}, func(b []byte) {
		readCh <- b
	})

	// Test write hook
	t0 := time.Now()
	rwct.Write([]byte("1234567890"))
	select {
	case b := <-writeCh:
		require.Equal(t, "1234567890", string(b))
		require.Less(t, time.Since(t0).Milliseconds(), int64(flushTimeout.Milliseconds()))
	case <-time.After(flushTimeout):
		t.Fatalf("data no flushed")
	}

	// Test read hook producing some data to read
	rwc.rx.Write([]byte("0987654321"))
	readData := make([]byte, 100)
	t0 = time.Now()
	n, err := rwct.Read(readData)
	require.NoError(t, err)
	require.Equal(t, "0987654321", string(readData[:n]))
	select {
	case b := <-readCh:
		require.Equal(t, "0987654321", string(b))
		require.Less(t, time.Since(t0).Milliseconds(), int64(flushTimeout.Milliseconds()))
	case <-time.After(flushTimeout):
		t.Fatalf("data no flushed")
	}

}

func TestReadHookTriggerFromWrite(t *testing.T) {
	rwc := newTestRWC()
	flushTimeout := time.Millisecond * 100
	readCh := make(chan []byte, 1)
	rwct := NewRWCTracer(rwc, 10, flushTimeout, nil, func(b []byte) {
		readCh <- b
	})

	// Produce some data to read and read it
	rwc.rx.Write([]byte("5678"))
	readData := make([]byte, 100)
	n, err := rwct.Read(readData)
	require.NoError(t, err)
	require.Equal(t, "5678", string(readData[:n]))

	// Write some data to trigger read hook immediately
	t0 := time.Now()
	rwct.Write([]byte("1234"))
	select {
	case b := <-readCh:
		require.Equal(t, "5678", string(b))
		require.Less(t, time.Since(t0).Milliseconds(), int64(flushTimeout.Milliseconds()))

	case <-time.After(flushTimeout):
		t.Fatalf("data no flushed")
	}
}

func TestWriteHookTriggerFromRead(t *testing.T) {
	rwc := newTestRWC()
	flushTimeout := time.Millisecond * 100
	writeCh := make(chan []byte, 1)
	rwct := NewRWCTracer(rwc, 10, flushTimeout, func(b []byte) {
		writeCh <- b
	}, nil)

	// Write some data
	rwct.Write([]byte("1234"))

	// Produce some data to read and read it.
	// This will trigger write hook immediately
	rwc.rx.Write([]byte("5678"))
	readData := make([]byte, 100)
	n, err := rwct.Read(readData)
	require.NoError(t, err)
	require.Equal(t, "5678", string(readData[:n]))
	t0 := time.Now()
	select {
	case b := <-writeCh:
		require.Equal(t, "1234", string(b))
		require.Less(t, time.Since(t0).Milliseconds(), int64(flushTimeout.Milliseconds()))

	case <-time.After(flushTimeout):
		t.Fatalf("data no flushed")
	}
}
