package buffer

import (
	"iter"
)

func NewRingBuffer[T any](n uint32) *RingBuffer[T] {
	capacity := powerOf2(n)
	return &RingBuffer[T]{
		s:       make([]T, capacity),
		cap:     capacity,
		modMask: capacity - 1, // = 2^n - 1
	}
}

type RingBuffer[T any] struct {
	s       []T
	cap     uint32
	modMask uint32
	start   uint32 // index of the beginning of the ring
	end     uint32 // index after last element of the ring
	full    bool
}

func (l *RingBuffer[T]) Full() bool { return l.full }

func (l *RingBuffer[T]) Empty() bool { return l.start == l.end && !l.full }

func (l *RingBuffer[T]) Capacity() int { return int(l.cap) }

func (l *RingBuffer[T]) Size() int {
	switch {
	case l.end < l.start:
		return int(l.cap - l.start + l.end)
	case l.end > l.start:
		return int(l.end - l.start)
	case l.full:
		return int(l.cap)
	default:
		return 0
	}
}

func (l *RingBuffer[T]) PushAll(s ...T) (n int) {
	for _, v := range s {
		if !l.Push(v) {
			break
		}
		n++
	}
	return
}

func (l *RingBuffer[T]) Push(v T) bool {
	if l.full {
		return false
	}
	slot := l.end
	l.s[slot] = v
	l.end = (l.end + 1) & l.modMask
	l.full = l.start == l.end
	return true
}

func (l *RingBuffer[T]) Pop() (T, bool) {
	if l.start == l.end && !l.full {
		var zero T
		return zero, false
	}
	v := l.s[l.start]
	l.start = (l.start + 1) & l.modMask
	l.full = false
	return v, true
}

func (l *RingBuffer[T]) Consume() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			v, ok := l.Pop()
			if !ok {
				break
			}
			if !yield(v) {
				return
			}
		}
	}
}

func powerOf2(v uint32) uint32 {
	// https://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2
	v--
	v |= v >> 1
	v |= v >> 2  //nolint:gomnd
	v |= v >> 4  //nolint:gomnd
	v |= v >> 8  //nolint:gomnd
	v |= v >> 16 //nolint:gomnd
	v++
	return v
}
