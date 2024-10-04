package nagle

import (
	"bytes"
	"context"
	"sync"
	"time"
)

type BytesHookFunc func([]byte)

type Nagle struct {
	cancel       context.CancelFunc
	buffer       *bytes.Buffer
	bufferSize   int
	flushTimeout time.Duration
	nagleHook    BytesHookFunc
	wg           sync.WaitGroup
	timer        *time.Timer
	mutex        sync.Mutex
}

func NewNagle(bufferSize int, flushTimeout time.Duration, nagleHook BytesHookFunc) *Nagle {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Nagle{
		cancel:       cancel,
		buffer:       &bytes.Buffer{},
		bufferSize:   bufferSize,
		flushTimeout: flushTimeout,
		nagleHook:    nagleHook,
		timer:        time.NewTimer(flushTimeout),
	}
	s.wg.Add(1)
	go s.flushWork(ctx)
	return s
}

func (s *Nagle) Write(data []byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.buffer.Write(data)

	if s.buffer.Len() >= s.bufferSize {
		s.flushLocked()
	}

	if !s.timer.Stop() {
		select {
		case <-s.timer.C:
		default:
		}
	}

	s.timer.Reset(s.flushTimeout)
}

func (s *Nagle) Flush() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.flushLocked()
}

func (s *Nagle) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.cancel()
	s.wg.Wait()
}

func (s *Nagle) flushWork(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-s.timer.C:
			s.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Nagle) flushLocked() {
	if s.buffer.Len() == 0 {
		return
	}
	s.nagleHook(s.buffer.Bytes())
	s.buffer.Reset()
}
