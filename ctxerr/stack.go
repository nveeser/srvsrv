package ctxerr

import (
	"fmt"
	"iter"
	"runtime"
)

// callers is a wrapper for runtime.Callers that allocates a slice.
func callers(skip int) stack {
	var stk [64]uintptr
	n := runtime.Callers(skip+2, stk[:])
	return stk[:n]
}

type stack []uintptr

func (s stack) frames() []*frame {
	var out []*frame
	frames := runtime.CallersFrames(s)
	var f runtime.Frame
	for i := 0; i < len(s); i++ {
		var ok bool
		f, ok = frames.Next()
		if !ok {
			break // Should never happen, and this is just debugging.
		}
		out = append(out, newFrame(i, f))
	}
	return out
}

var newFrame = func(i int, f runtime.Frame) *frame {
	return &frame{
		file:     f.File,
		line:     f.Line,
		funcName: f.Func.Name(),
	}
}

// walkStack returns a sequence of frames from the specified stack and stops when
// the frame matches the frame of the caller when the frames match the same frame
// of the caller.
func walkStack(s stack, skip int) iter.Seq[*frame] {
	return func(yield func(*frame) bool) {
		stackFrames := s.frames()
		callerFrames := callers(skip + 1).frames()
		for i, sf := range stackFrames {
			end := len(stackFrames) - i
			j := len(callerFrames) - end
			if j > 0 {
				cf := callerFrames[j]
				if cf.funcName == sf.funcName {
					return
				}
			}
			if !yield(sf) {
				return
			}
		}
	}
}

type frame struct {
	file     string
	line     int
	funcName string
}

func (f *frame) String() string {
	return fmt.Sprintf("[%s:%d] %s", f.file, f.line, f.funcName)
}
