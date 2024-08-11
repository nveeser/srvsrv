package buffer

import (
	"cmp"
	"iter"
	"slices"
)

func compare[T Numbered](a, b T) int {
	return cmp.Compare(b.Seq(), b.Seq())
}
func compareN[T Numbered](v T, n int32) int {
	return cmp.Compare(v.Seq(), n)
}

type Numbered interface {
	Seq() int32
}

type SliceBuffer[T Numbered] struct {
	s    []T
	mark int32
}

func (l *SliceBuffer[T]) Add(n int32, v T) {
	if _, ok := l.Find(n); ok {
		panic("block already exists")
	}
	l.s = append(l.s)
	slices.SortFunc(l.s, compare[T])
}

func (l *SliceBuffer[T]) Remove(n int32) (T, bool) {
	ix, ok := slices.BinarySearchFunc(l.s, n, compareN[T])
	if !ok {
		var zero T
		return zero, false
	}
	v := l.s[ix]
	l.s = slices.Delete(l.s, ix, ix+1)
	return v, true
}

func (l *SliceBuffer[T]) Find(n int32) (T, bool) {
	ix, ok := slices.BinarySearchFunc(l.s, n, compareN[T])
	if !ok {
		var zero T
		return zero, false
	}
	return l.s[ix], true
}

func (l *SliceBuffer[T]) sequentialBlocks() iter.Seq[T] {
	return func(yield func(T) bool) {
		var consumed int
		for i, blk := range l.s {
			if blk.Seq() != l.mark {
				break
			}
			l.mark++
			consumed = i
			if !yield(blk) {
				return
			}
		}
		l.s = slices.Delete(l.s, 0, consumed)
	}
}
