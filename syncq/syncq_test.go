package syncq

import (
	"context"
	"errors"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"
)

func ExampleQueue_PushPop(t *testing.T) {
	ctx := context.Background()
	q := New[int]()
	defer q.WaitEmpty(ctx)

	if err := q.Push(ctx, 3); err != nil {
		t.Errorf("Push() got error: %s", err)
	}
	q.Close() // No more calls to Push()

	got, open := q.Pop(ctx)
	if !open {
		t.Errorf("Pop() got closed")
	}

	want := 3
	if got != want {
		t.Errorf("got %d want %d", got, want)
	}
}

func TestPush(t *testing.T) {
	t.Run("ErrorOnCanceledContext", func(t *testing.T) {
		ctx := context.Background()
		q := New[int]()
		defer q.WaitEmpty(context.Background())

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		err := q.Push(ctx, 1)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Push got err %v wanted err %v", err, context.Canceled)
		}
	})
	t.Run("ErrorShutdownQueue", func(t *testing.T) {
		ctx := context.Background()
		q := New[int]()
		defer q.WaitEmpty(context.Background())
		q.Shutdown()

		err := q.Push(ctx, 1)
		if !errors.Is(err, ErrQueueShutdown) {
			t.Errorf("Push got err %v wanted err %v", err, ErrQueueShutdown)
		}
	})
}

func TestPop(t *testing.T) {
	var value int
	var found bool
	var done chan any

	goPop := func(ctx context.Context, queue *Queue[int]) {
		done = make(chan any)
		go func() {
			value, found = queue.Pop(ctx)
			close(done)
		}()
	}
	isBlocked := func() bool {
		select {
		case <-done:
			return false
		default:
			return true
		}
	}
	hasValue := func(d time.Duration) bool {
		select {
		case <-done:
			return true
		case <-time.After(d):
			return false
		}
	}

	t.Run("BlockUntilValue", func(t *testing.T) {
		ctx := context.Background()
		q := New[int]()
		defer q.WaitEmpty(ctx)

		goPop(ctx, q)
		if !isBlocked() {
			t.Errorf("Pop() did not block")
		}

		if err := q.Push(ctx, 3); err != nil {
			t.Errorf("Push() got error: %s", err)
		}

		if !hasValue(500 * time.Millisecond) {
			t.Errorf("timeout wanting for Pop() to return")
		}
		if value != 3 || found != true {
			t.Errorf("Pop() got (%d, %t) wanted (%d, %t)", value, found, 3, false)
		}
	})
	t.Run("BlockUntilClose", func(t *testing.T) {
		ctx := context.Background()
		q := New[int]()
		defer q.WaitEmpty(ctx)

		goPop(ctx, q)
		if !isBlocked() {
			t.Errorf("Pop() did not block")
		}

		q.Close()

		if !hasValue(500 * time.Millisecond) {
			t.Errorf("timeout wanting for Pop()")
		}
		if value != 0 || found != false {
			t.Errorf("Pop() got (%d, %t) wanted (%d, %t)", value, found, 0, false)
		}
	})
	t.Run("BlockUntilCancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		q := New[int]()
		defer q.WaitEmpty(context.Background())

		goPop(ctx, q)
		if !isBlocked() {
			t.Errorf("Pop() did not block")
		}

		cancel()

		if !hasValue(500 * time.Millisecond) {
			t.Errorf("timeout wanting for Pop()")
		}
		if value != 0 || found != false {
			t.Errorf("Pop() got (%d, %t) wanted (%d, %t)", value, found, 0, false)
		}
	})
}

func TestWaitEmpty(t *testing.T) {
	t.Run("Pop/empty=true", func(t *testing.T) {
		ctx := context.Background()
		q := New[int]()
		q.Push(ctx, 3)
		q.Close()

		done := make(chan any)
		var gotEmpty bool
		go func() {
			gotEmpty = q.WaitEmpty(ctx)
			close(done)
		}()

		q.Pop(ctx)

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			t.Errorf("WaitEmpty() did not hasValue")
		}

		if gotEmpty != true {
			t.Errorf("WaitEmpty() got %t wanted %t", gotEmpty, true)
		}
	})
	t.Run("ContextCanceled/empty=false", func(t *testing.T) {
		ctx := context.Background()
		q := New[int]()
		q.Push(ctx, 3)
		q.Close()
		dctx, cancel := context.WithCancel(ctx)

		done := make(chan any)
		var gotEmpty bool
		go func() {
			gotEmpty = q.WaitEmpty(dctx)
			close(done)
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			t.Errorf("WaitEmpty() did not hasValue")
		}

		if gotEmpty != false {
			t.Errorf("WaitEmpty() got %t wanted %t", gotEmpty, false)
		}
	})
}

func TestSize(t *testing.T) {
	ctx := context.Background()
	q := New[int]()
	defer q.WaitEmpty(ctx)
	for i := 0; i < 10; i++ {
		q.Push(ctx, i)
	}
	n, m := q.Size()
	if n != 10 || m != 10 {
		t.Errorf("Size got (%d, %d) want (%d, %d)", n, m, 10, 10)
	}
	q.Close()
	for {
		_, ok := q.Pop(ctx)
		if !ok {
			break
		}
	}
	n, m = q.Size()
	if n != 0 || m != 10 {
		t.Errorf("Size got (%d, %d) want (%d, %d)", n, m, 0, 10)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	cases := []struct {
		name      string
		producers *producers
		consumers *consumers
		want      []int
		wantErr   bool
	}{
		{
			name:      "push=1/pop=1",
			producers: &producers{n: 1},
			consumers: &consumers{n: 1},
			want:      want(10, 1),
		},
		{
			name:      "push=3/pop=1",
			producers: &producers{n: 3},
			consumers: &consumers{n: 1},
			want:      want(10, 3),
		},
		{
			name:      "push=1/pop=3",
			producers: &producers{n: 1},
			consumers: &consumers{n: 3},
			want:      want(10, 1),
		},
		{
			name:      "push=3/pop=3",
			producers: &producers{n: 3},
			consumers: &consumers{n: 3},
			want:      want(10, 3),
		},
		{
			name:      "push=10/pop=10",
			producers: &producers{n: 10, writes: 100},
			consumers: &consumers{n: 10},
			want:      want(100, 10),
		},
		{
			name: "push=err/pop=10",
			producers: &producers{
				n:      10,
				writes: 10,
				errFn:  makeErrFn(3),
			},
			consumers: &consumers{n: 10},
			wantErr:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			q := New[int]()
			defer q.WaitEmpty(ctx)

			tc.producers.Go(ctx, q)
			tc.consumers.Go(ctx, q)

			var g errgroup.Group
			g.Go(func() error {
				defer q.Close()
				return tc.producers.Wait()
			})
			g.Go(func() error {
				if err := tc.consumers.Wait(); err != nil {
					q.Shutdown()
					return err
				}
				return nil
			})
			gotErr := g.Wait()
			switch {
			case gotErr != nil && !tc.wantErr:
				t.Errorf("producers/consumers returned an error %s wanted nil", gotErr)
			case gotErr == nil && tc.wantErr:
				t.Errorf("producers/consumers returned an error nil, wanted non-nil")
			}

			if !tc.wantErr {
				sort.Ints(tc.want)
				got := tc.consumers.Got()
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("Got diff -want, +got: %s", diff)
				}
			}
		})
	}
}

// want returns a slice of ints in the range of [0, max) with each value added n
// times.
func want(max, n int) (out []int) {
	for i := 0; i < max; i++ {
		for j := 0; j < n; j++ {
			out = append(out, i)
		}
	}
	sort.Ints(out)
	return
}

func makeErrFn(after int) func() error {
	return func() error {
		if after > 0 {
			after--
			return nil
		}
		return errors.New("fake error")
	}
}

type producers struct {
	writes int
	n      int
	g      *errgroup.Group
	errFn  func() error
}

func (w *producers) Go(ctx context.Context, q *Queue[int]) {
	if w.writes == 0 {
		w.writes = 10
	}
	errFunc := w.errFn
	if errFunc == nil {
		errFunc = func() error { return nil }
	}
	w.g, ctx = errgroup.WithContext(ctx)
	for i := 0; i < w.n; i++ {
		w.g.Go(func() error {
			for j := 0; j < w.writes; j++ {
				if err := q.Push(ctx, j); err != nil {
					return err
				}
				if err := errFunc(); err != nil {
					return err
				}
			}
			return nil
		})
	}
}

func (w *producers) Wait() error { return w.g.Wait() }

type consumers struct {
	n     int
	got   []int
	mu    sync.Mutex
	g     *errgroup.Group
	errFn func() error
}

func (w *consumers) Got() []int {
	w.mu.Lock()
	got := slices.Clone(w.got)
	w.mu.Unlock()
	sort.Ints(got)
	return got
}

func (w *consumers) Go(ctx context.Context, q *Queue[int]) {
	errFunc := w.errFn
	if errFunc == nil {
		errFunc = func() error { return nil }
	}
	w.g, ctx = errgroup.WithContext(ctx)
	for i := 0; i < w.n; i++ {
		w.g.Go(func() error {
			for {
				v, isValue := q.Pop(ctx)
				if !isValue {
					return nil
				}
				if err := errFunc(); err != nil {
					return err
				}
				w.mu.Lock()
				w.got = append(w.got, v)
				w.mu.Unlock()
			}
		})
	}
}

func (w *consumers) Wait() error { return w.g.Wait() }
