package ctxerr

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

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
		thisFrame := callerFrame(e.stack.callers, i)
		name := thisFrame.funcName
		if !diff && i < len(walkerStack.callers) {
			cFrame := callerFrame(walkerStack.callers, i)
			if name == cFrame.funcName {
				// both stacks share this PC, skip it.
				continue
			}
			diff = true
		}
		if name == prev {
			continue
		}
		// TODO - consider re-enabling trimming to keep file paths cleaner
		// name, ok := trimPrev(prev, name)
		f(thisFrame.file, thisFrame.line, name)
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

type frame struct {
	file     string
	line     int
	funcName string
}

func (f *frame) String() string {
	return fmt.Sprintf("[%s:%d] %s", f.file, f.line, f.funcName)
}

// frame returns the nth frame, with the frame at top of stack being 0.
var callerFrame = func(callers []uintptr, n int) *frame {
	frames := runtime.CallersFrames(callers)
	var f runtime.Frame
	for i := len(callers) - 1; i >= n; i-- {
		var ok bool
		f, ok = frames.Next()
		if !ok {
			break // Should never happen, and this is just debugging.
		}
	}
	return &frame{
		file:     f.File,
		line:     f.Line,
		funcName: f.Func.Name(),
	}
}

// callers is a wrapper for runtime.Callers that allocates a slice.
func callers(skip int) stack {
	var stk [64]uintptr
	n := runtime.Callers(skip, stk[:])
	return stack{stk[:n]}
}
