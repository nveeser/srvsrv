// Package syncq provides a concurrent Queue with channel semantics.
package syncq

import (
	"context"
	"errors"
	"fmt"
	"github.com/nveeser/srvsrv/ctxerr"
	"strings"
	"sync/atomic"
)

// Queue provides synchronous queue for one or more concurrent providers and one
// or more consumers.
//
// One or more concurrent callers adds data using Push(). One or more concurrent
// consumers receive values using Pop(). Pop() will block until a value is available,
// the Queue is closed (and empty) or the provided context is canceled.
//
// Similar to a pipeline pattern with channels, when there are no more values to
// add, Close() should be called. Once the Queue is empty and closed, all calls
// to Pop() will return a zero value and false. If there are no more consumers
// to call Pop() then Cancel().
//
// # Summary
//
//   - Producers call Push()
//   - Consumers call Pop()
//   - Close() signals to consumers that Push() will no longer be called
//   - Shutdown() signals to producers that Pop() will no longer be called
type Queue[E any] struct {
	max      int
	pushc    chan E
	popc     chan E
	shutdown chan any
	done     chan any
	size     atomic.Int64
	total    atomic.Int64
	full     atomic.Int64
}

// New returns a new initialized Queue. The caller is responsible for calling
// Close once now more elements are to be written.
//
// To ensure that no resources are leaked it is common to defer a call to
// WaitEmpty() to ensure that the internal goroutine completes.
func New[E any](n int) *Queue[E] {
	q := &Queue[E]{
		max:      n,
		pushc:    make(chan E),
		popc:     make(chan E),
		shutdown: make(chan any),
		done:     make(chan any),
	}
	go q.goqueue()
	return q
}

// Size returns the current number of elements in the
// the queue followed by the total number of elements that
// have been processed by the queue.
func (q *Queue[E]) Size() (size, total int64) {
	s, t := q.size.Load(), q.total.Load()
	return s, t
}

var ErrQueueShutdown = errors.New("Queue is shutdown")

// Push adds the specified value to the queue. If the context expires before the
// value can be enqueued then an error is returned. If the queue has been
// canceled the queue returns ErrQueueCanceled. Calling Push() after
// Close() will panic.
func (q *Queue[E]) Push(ctx context.Context, e E) error {
	logf("Push: start")
	select {
	case <-ctx.Done():
		logf("Push: context.Canceled")
		return ctxerr.E(ctx, ctx.Err())
	case <-q.shutdown:
		logf("Push: q.shutdown")
		return ErrQueueShutdown
	case q.pushc <- e:
		logf("Push: add element")
		q.total.Add(1)
		q.size.Add(1)
		return nil
	}
}

// Pop returns the next item in the queue. If no item is available the call
// blocks until an item is available. If the Queue is closed and empty, or canceled or the
// specified context expires then the zero value and false is returned.
func (q *Queue[E]) Pop(ctx context.Context) (element E, open bool) {
	logf("Pop: start")
	var zero E
	select {
	case x, found := <-q.popc:
		if found {
			q.size.Add(-1)
		}
		logf("Pop: got")
		return x, found

	case <-q.shutdown:
		logf("Pop: q.shutdown")
		return zero, false
	case <-ctx.Done():
		logf("Pop: ctx.Canceled")
		return zero, false
	}
}

// Close marks the Queue as closed and signals that no more elements
// are going to be added. Any calls to Push() after the queue is
// closed will return an error
func (q *Queue[E]) Close() { closeOnce(q.pushc) }

// Shutdown shuts down the queue. After shutdown all calls
// to Push() will return a ErrQueueShutdown and all calls to Pop()
// will return zero value and false
func (q *Queue[E]) Shutdown() { closeOnce(q.shutdown) }

// WaitEmpty blocks until the Queue is empty or the context
// is canceled. If the context is canceled the queue is shutdown
// and any remaining values are not guaranteed to be processed.
func (q *Queue[E]) WaitEmpty(ctx context.Context) bool {
	q.Close()
	select {
	case <-q.done:
		logf("WaitEmpty: q.done")
		return true
	case <-ctx.Done():
		logf("WaitEmpty: q.shutdown")
		q.Shutdown()
		<-q.done
		logf("WaitEmpty: q.done")
		return false
	}
}

func closeOnce[E any](c chan E) {
	select {
	case <-c:
		return
	default:
		close(c)
	}
}

func (q *Queue[E]) goqueue() {
	defer close(q.done)
	defer close(q.popc)

	var s state[E]

	for {
		qEmpty := len(s.queue) == 0
		qReady := len(s.queue) > 0
		qFull := len(s.queue) >= q.max && q.max > 0

		switch {
		case !s.closed && !qFull && s.pushc == nil:
			logf("goqueue: full=false %s", &s)
			s.pushc = q.pushc
		case !s.closed && qFull && s.pushc != nil:
			logf("goqueue: full=true %s", &s)
			q.full.Add(1)
			s.pushc = nil
		// input channel is closed and queue empty
		case s.closed && qEmpty:
			logf("goqueue: closed empty %s", &s)
			return
		}
		// output channel is ready / queue not empty
		if qReady && s.popc == nil {
			logf("goqueue: set next")
			s.next = s.queue[0]
			s.popc = q.popc
		}

		logf("goqueue: select %s", &s)
		select {
		case e, ok := <-s.pushc:
			if ok {
				logf("goqueue: pushc -> queue %s", &s)
				s.queue = append(s.queue, e)
			} else {
				logf("goqueue: pushc -> closed %s", &s)
				s.pushc = nil
				s.closed = true
			}

		case s.popc <- s.next:
			logf("goqueue: popc <- next %s", &s)
			s.queue = s.queue[1:]
			s.popc = nil

		case <-q.shutdown:
			logf("goqueue: shutdown")
			return
		}
	}
}

type state[E any] struct {
	pushc  chan E // nil when queue is closed or full
	popc   chan E // non-nil when value is ready to send
	closed bool
	queue  []E
	next   E
}

func (s *state[E]) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "pushc=")
	if s.pushc == nil {
		fmt.Fprintf(&b, "off ")
	} else {
		fmt.Fprintf(&b, "on ")
	}
	fmt.Fprintf(&b, "popc=")
	if s.popc == nil {
		fmt.Fprintf(&b, "off ")
	} else {
		fmt.Fprintf(&b, "on ")
	}
	fmt.Fprintf(&b, "closed=%t ", s.closed)
	fmt.Fprintf(&b, "queue=%d", len(s.queue))
	return b.String()
}

// TODO figure out if this is useful.

type BatchingQueue[E any] struct {
	*Queue[[]E]
	n int
}

// Push adds the specified value to the queue. If the context expires before the
// value can be enqueued then an error is returned. If the queue has been
// canceled the queue returns ErrQueueCanceled. Calling Push() after
// Close() will panic.
func (q *BatchingQueue[E]) Push(ctx context.Context, e ...E) error {
	return q.Queue.Push(ctx, e)
}

// Pop returns the next item in the queue. If no item is available the call
// blocks until an item is available. If the Queue is closed and empty, or canceled or the
// specified context expires then the zero value and false is returned.
func (q *BatchingQueue[E]) Pop(ctx context.Context) (element []E, open bool) {
	return q.Queue.Pop(ctx)
}

type buffer[E any] []E

func (b *buffer[E]) add(e E)   { *b = append(*b, e) }
func (b *buffer[E]) size() int { return len(*b) }
func (b *buffer[E]) next() (E, bool) {
	var next E
	var queue = *b
	if len(queue) == 0 {
		return next, false
	}
	next, *b = queue[0], queue[1:]
	return next, true
}

type batchingBuffer[E any, S ~[]E] struct {
	queue []E
	n     int
}

func (b *batchingBuffer[E, S]) add(e S)   { b.queue = append(b.queue, e...) }
func (b *batchingBuffer[E, S]) size() int { return len(b.queue) }
func (b *batchingBuffer[E, S]) next() (S, bool) {
	var next S
	if len(b.queue) < b.n {
		return next, false
	}
	next, b.queue = b.queue[:b.n], b.queue[b.n:]
	return next, true
}

var logf = func(msg string, args ...any) {}
