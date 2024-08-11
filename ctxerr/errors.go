// Package ctxerr is an experiment for how to add
// context to errors.
// There are three different mechanisms here
// context - which
// stack - capture the full go stack at the first sign
// operation - this is context added by callers "as needed" which can approximate a call stack but
package ctxerr

import (
	"context"
	"errors"
	"fmt"
	"github.com/cyrusaf/ctxlog"
	"io"
	"log"
	"log/slog"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

var ErrorPrefixKeys = []string{
	"module",
	"call",
	"action",
	"error",
}

type Op string

type Error struct {
	Op    Op
	Msg   string
	Err   error
	Attrs map[string]string
	stack
}

func E(args ...any) error {
	e := newError(args...)
	return e
}

// Tests:
// 0 args
// 1 args
// 2 args
// no string

func Ef(args ...any) error {
	idx := slices.IndexFunc(args, func(v any) bool {
		_, ok := v.(string)
		return ok
	})
	if idx > 0 {
		var fmtArgs []any
		args, fmtArgs = args[:idx], args[idx:]
		msg := fmtArgs[0].(string)
		if len(fmtArgs) > 1 {
			msg = fmt.Sprintf(msg, fmtArgs[1:]...)
		}
		args = append(args, msg)
	}
	e := newError(args...)
	return e
}

func newError(args ...any) *Error {
	e := &Error{}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Op:
			e.Op = arg

		case *Error:
			// Make a copy
			copyArg := *arg
			e.Err = &copyArg

		case context.Context:
			v := ctxlog.GetAttrs(arg)
			e.Attrs = make(map[string]string)
			for _, attr := range v {
				e.Attrs[attr.Key] = attr.Value.String()
			}

		case error:
			if arg == nil {
				panic("nil error passed to E()")
			}
			e.Err = arg
		case string:
			e.Msg = arg

		default:
			_, file, line, _ := runtime.Caller(1)
			log.Printf("errors.E: bad call from %s:%d: %v", file, line, args)
			panic("E() called with unknown type" + reflect.TypeOf(arg).String())
		}
	}
	e.populateStack()
	return e
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) isZero() bool {
	return e.Op == "" && e.Msg == "" && e.Err == nil
}

func (e *Error) Error() string {
	var b strings.Builder
	e.writeSummary(&b, true)
	return b.String()
}

func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'w', 's':
		e.writeSummary(s, true)

	case 'v':
		var curr error = e
		var stacked *Error = e
		var written bool
		for curr != nil {
			if written {
				io.WriteString(s, "\n")
			}
			if ee, ok := curr.(*Error); ok {
				ee.writeSummary(s, false)
				stacked = ee
			} else {
				io.WriteString(s, curr.Error())
			}
			written = true
			curr = errors.Unwrap(curr)
		}
		stacked.walkStack(3, func(file string, line int, fname string) {
			if written {
				io.WriteString(s, "\n\t")
			}
			writeCallsite(s, file, line)
			io.WriteString(s, " \n\t   ")
			io.WriteString(s, fname)
			io.WriteString(s, "(...)")
			written = true
		})
	}
}

var writeCallsite = func(w io.Writer, file string, line int) {
	w.Write([]byte(file))
	w.Write([]byte(":"))
	w.Write(strconv.AppendInt(nil, int64(line), 10))
}

func (e *Error) writeSummary(w io.Writer, withCause bool) {
	var written bool
	if e.Op != "" {
		io.WriteString(w, "[")
		io.WriteString(w, string(e.Op))
		io.WriteString(w, "] ")
		written = true
	}
	if e.Msg != "" {
		if written {
			io.WriteString(w, ": ")
		}
		io.WriteString(w, e.Msg)
	}
	if e.Err != nil && withCause {
		if ee, ok := e.Err.(*Error); ok {
			ee.writeSummary(w, false)
		} else {
			if !written {
				io.WriteString(w, "ctxerr.Error")
			}
			io.WriteString(w, ": ")
			io.WriteString(w, e.Err.Error())
		}
	}
}

func ContextError(ctx context.Context, err error) error {
	var format strings.Builder
	args := buildPrefixFormat(&format, ctxlog.GetAttrs(ctx))

	args = append(args, err)
	format.WriteString(": %w")
	return fmt.Errorf(format.String(), args)
}

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

// Errorf returns a new error adding attributes from the context
// using ctxerr.
func Errorf(ctx context.Context, format string, args ...any) error {
	var prefix strings.Builder
	xargs := buildPrefixFormat(&prefix, ctxlog.GetAttrs(ctx))
	prefix.WriteString(format)
	format = prefix.String()
	args = append(xargs, args...)
	return fmt.Errorf(format, args...)
}

func buildPrefixFormat(b *strings.Builder, attrs []slog.Attr) []any {
	var args []any
	for _, attr := range attrs {
		if slices.Contains(ErrorPrefixKeys, attr.Key) {
			switch attr.Value.Any().(type) {
			case string, fmt.Stringer:
				b.WriteString("[%s]")
			default:
				panic("context key is not an string or fmt.Stringer: " + attr.Key)
			}
			args = append(args, attr)
		}
	}
	return args
}
