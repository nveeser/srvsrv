package buffer

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"slices"
	"testing"
)

func TestRingQueue(t *testing.T) {
	t.Run("Push-Pop", func(t *testing.T) {
		ring := NewRingBuffer[int](4)
		if !ring.Push(8) {
			t.Errorf("Push() got false wanted true")
		}
		got, ok := ring.Pop()
		if !ok {
			t.Errorf("Pop() got ok=false wanted ok=true")
		}
		if got != 8 {
			t.Errorf("Pop() got %d wanted %d", got, 8)
		}
	})
	t.Run("PushAll-Consume", func(t *testing.T) {
		ring := NewRingBuffer[int](4)
		if n := ring.PushAll(3, 4, 5, 6); n != 4 {
			t.Errorf("PushAll() got %d wanted %d", n, 4)
		}
		got := slices.Collect(ring.Consume())
		want := []int{3, 4, 5, 6}
		if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Consume() got diff -want/+got: %s", diff)
		}
	})
	t.Run("Empty", func(t *testing.T) {
		ring := NewRingBuffer[int](3)
		for i := 0; i < 5; i++ {
			if !ring.Empty() {
				t.Errorf("Empty() got false wanted true")
			}
			_ = ring.Push(3)
			if ring.Empty() {
				t.Errorf("Empty() got true wanted false")
			}
			ring.Pop()
		}
	})
	t.Run("Full", func(t *testing.T) {
		ring := NewRingBuffer[int](4)
		if n := ring.PushAll(3, 4, 5, 6); n != 4 {
			t.Errorf("PushAll() got %d wanted %d", n, 4)
		}
		for i := 0; i < 5; i++ {
			if !ring.Full() {
				t.Errorf("Full() got false wanted true")
			}
			ring.Pop()
			if ring.Full() {
				t.Errorf("Empty() got true wanted false")
			}
			ring.Push(3)
		}
	})
}
