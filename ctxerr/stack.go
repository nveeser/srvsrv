package ctxerr

import (
	"errors"
	"runtime"
	"strings"
)

var fakeLineNumbers = false

type stack struct {
	callers []uintptr
}

// populateStack will update the callers if there is no existing
// Error value found by unwrapping this error with errors.As().
func (e *Error) populateStack() {
	var e2 *Error
	// only if there is no *Error value in the cause chain.
	if !errors.As(e.Err, &e2) {
		e.stack = callers(5)
	}
}

type stackFn func(file string, line int, fname string)

func (e *Error) walkStack(skip int, f stackFn) {
	walkerStack := callers(skip)
	var prev string // the name of the last-seen function
	var diff bool   // true after the two stacks diverge
	for i := 0; i < len(e.stack.callers); i++ {
		thisFrame := frame(e.stack.callers, i)
		name := thisFrame.Func.Name()

		if !diff && i < len(walkerStack.callers) {
			if name == frame(walkerStack.callers, i).Func.Name() {
				// both stacks share this PC, skip it.
				continue
			}
			diff = true
		}
		if name == prev {
			continue
		}
		//name, ok := trimPrev(prev, name)
		line := thisFrame.Line
		if fakeLineNumbers {
			line = 0
		}
		f(thisFrame.File, line, name)
		prev = name
	}
}

func trimPrev(prev, next string) (string, bool) {
	// Find the uncommon prefix between this and the previous
	// function name, separating by dots and slashes.
	trim := 0
	for {
		j := strings.IndexAny(next[trim:], "./")
		if j < 0 {
			break
		}
		if !strings.HasPrefix(prev, next[:j+trim]) {
			break
		}
		trim += j + 1 // skip over the separator
	}
	return next[trim:], trim > 0
}

// frame returns the nth frame, with the frame at top of stack being 0.
func frame(callers []uintptr, n int) *runtime.Frame {
	frames := runtime.CallersFrames(callers)
	var f runtime.Frame
	for i := len(callers) - 1; i >= n; i-- {
		var ok bool
		f, ok = frames.Next()
		if !ok {
			break // Should never happen, and this is just debugging.
		}
	}
	return &f
}

// callers is a wrapper for runtime.Callers that allocates a slice.
func callers(skip int) stack {
	var stk [64]uintptr
	n := runtime.Callers(skip, stk[:])
	return stack{stk[:n]}
}

//func callsite(depth int) string {
//	_, file, line, ok := runtime.Caller(depth)
//	if !ok {
//		return "<unknown:???>"
//	}
//	return fmt.Sprintf("%s:%d", file, line)
//}
//
//func stackTrace() {
//	var pcs [32]uintptr
//	n := runtime.Callers(3, pcs[:])
//	frames := runtime.CallersFrames(pcs[:n])
//	for {
//		f, more := frames.Next()
//		fmt.Printf("%s:%d (%s)\n", f.File, f.Line, f.Function)
//		if !more {
//			break
//		}
//	}
//}
