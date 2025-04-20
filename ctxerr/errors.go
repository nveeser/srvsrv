// Package ctxerr is an experiment for how to add
// context to errors.
// There are three different mechanisms here
// context - which
// stack - capture the full go stack at the first sign
// operation - this is context added by callers "as needed" which can approximate a call stack but
package ctxerr

import (
	"context"
	"fmt"
	"github.com/cyrusaf/ctxlog"
	"log"
	"log/slog"
	"reflect"
	"runtime"
	"slices"
	"strings"
)

var ErrorPrefixKeys = []string{
	"module",
	"call",
	"action",
	"error",
}

type Op string

func Opf(format string, args ...any) Op {
	return Op(fmt.Sprintf(format, args...))
}

type Error struct {
	Op    Op
	Msg   string
	Cause error
	Attrs map[string]string
	stack
}

func E(args ...any) error {
	if len(args) == 0 {
		panic("E() called with no arguments")
	}
	e := newError(1, args)
	return e
}

func Ef(args ...any) error {
	if len(args) == 0 {
		panic("E() called with no arguments")
	}
	e := newError(1, formatStringArgs(args))
	return e
}

func formatStringArgs(args []any) []any {
	idx := slices.IndexFunc(args, func(v any) bool {
		_, ok := v.(string)
		return ok
	})
	if idx < 0 {
		return args
	}
	var fmtArgs []any
	args, fmtArgs = args[:idx], args[idx:]
	msg := fmtArgs[0].(string)
	if len(fmtArgs) > 1 {
		msg = fmt.Sprintf(msg, fmtArgs[1:]...)
	}
	return append(args, msg)
}

func newError(skip int, args []any) *Error {
	e := &Error{
		stack: callers(skip + 1),
	}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Op:
			e.Op = arg

		case *Error:
			// Make a copy
			copyArg := *arg
			e.Cause = &copyArg

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
			e.Cause = arg
		case string:
			e.Msg = arg

		default:
			_, file, line, _ := runtime.Caller(1)
			log.Printf("errors.E: bad call from %s:%d: %v", file, line, args)
			panic("E() called with unknown type" + reflect.TypeOf(arg).String())
		}
	}

	if e.Op == "" && e.Cause == nil && e.Msg == "" && e.Attrs == nil {
		panic("E() called with no arguments")
	}
	return e
}

func (e *Error) Unwrap() error { return e.Cause }

func (e *Error) isZero() bool {
	return e.Op == "" && e.Msg == "" && e.Cause == nil
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s", e)
}

func ContextError(ctx context.Context, err error) error {
	var format strings.Builder
	args := buildPrefixFormat(&format, ctxlog.GetAttrs(ctx))

	args = append(args, err)
	format.WriteString(": %w")
	return fmt.Errorf(format.String(), args)
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
