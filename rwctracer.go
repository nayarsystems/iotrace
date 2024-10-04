package iotrace

import (
	"github.com/nayarsystems/iotrace/nagle"
	"io"
	"sync"
	"time"
)

type RWCTracer struct {
	rwc     io.ReadWriteCloser
	txNagle *nagle.Nagle
	rxNagle *nagle.Nagle
	closed  bool
	mutex   sync.Mutex
}

func NewRWCTracer(rwc io.ReadWriteCloser, bufferSize int, flushTimeout time.Duration, txHook, rxHook func([]byte)) *RWCTracer {
	var txNagle *nagle.Nagle
	var rxNagle *nagle.Nagle
	if txHook != nil {
		txNagle = nagle.NewNagle(bufferSize, flushTimeout, txHook)
	}
	if rxHook != nil {
		rxNagle = nagle.NewNagle(bufferSize, flushTimeout, rxHook)
	}
	t := &RWCTracer{
		rwc:     rwc,
		txNagle: txNagle,
		rxNagle: rxNagle,
	}
	return t
}

func (t *RWCTracer) Close() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return io.ErrClosedPipe
	}

	e := t.rwc.Close()
	t.closed = true

	if t.txNagle != nil {
		t.txNagle.Close()
	}

	if t.rxNagle != nil {
		t.rxNagle.Close()
	}

	return e
}

func (t *RWCTracer) Write(data []byte) (n int, e error) {
	n, e = t.rwc.Write(data)

	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return 0, io.ErrClosedPipe
	}

	if n > 0 {
		if t.rxNagle != nil {
			t.rxNagle.Flush()
		}
		if t.txNagle != nil {
			t.txNagle.Write(data[:n])
		}
	}
	return
}

func (t *RWCTracer) Read(data []byte) (n int, e error) {
	n, e = t.rwc.Read(data)

	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.closed {
		return 0, io.ErrClosedPipe
	}

	if n > 0 {
		if t.txNagle != nil {
			t.txNagle.Flush()
		}
		if t.rxNagle != nil {
			t.rxNagle.Write(data[:n])
		}
	}
	return
}
