package ctxerr

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"
)

const tab = "   "

func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 's', 'v':
		writeSummary(s, e, true)
	case 'x':
		var indent string
		if width, ok := s.Width(); ok {
			indent = strings.Repeat(tab, width)
		}
		stackErr := writeChain(s, indent, e)
		if s.Flag('+') {
			io.WriteString(s, "\n")
			io.WriteString(s, indent)
			io.WriteString(s, "[Inner Stack]\n")
			writeStack(s, indent+tab, stackErr)
		}
	}
}

// writeSummary writes a single line summary of the Error. If withCause it true
// and the Error contains a cause, then the output of that error is written as
// well.
func writeSummary(w io.Writer, e *Error, withCause bool) {
	ce := e
	var written bool
	for ce != nil {
		if ce.Op != "" {
			if written {
				io.WriteString(w, " ")
			}
			io.WriteString(w, "[")
			io.WriteString(w, string(ce.Op))
			io.WriteString(w, "]")
			written = true
		}
		if ce.Msg != "" {
			if written {
				io.WriteString(w, ": ")
			}
			io.WriteString(w, ce.Msg)
		}
		if !withCause || ce.Cause == nil {
			return
		}
		if ne, ok := ce.Cause.(*Error); ok {
			ce = ne
			continue
		}
		io.WriteString(w, ": ")
		fmt.Fprintf(w, "%v", ce.Cause)
		return
	}
}

func writeChain(w io.Writer, indent string, err *Error) (last *Error) {
	var written bool
	for curr := range unwrap(err) {
		if written {
			io.WriteString(w, "\n")
		}
		io.WriteString(w, indent)
		switch xe := curr.(type) {
		case *Error:
			writeSummary(w, xe, false)
			if f := xe.stack.frames(); len(f) > 0 {
				io.WriteString(w, "\n")
				io.WriteString(w, indent)
				io.WriteString(w, tab)
				writeCallsite(w, f[0])
			}
			last = xe
		default:
			fmt.Fprintf(w, "%s", xe)
		}
		written = true
	}
	return last
}

func writeStack(s io.Writer, indent string, e *Error) {
	var written bool
	for frame := range walkStack(e.stack, 1) {
		if written {
			io.WriteString(s, "\n")
		}
		io.WriteString(s, indent)
		writeCallsite(s, frame)
		io.WriteString(s, "\n")
		io.WriteString(s, indent)
		io.WriteString(s, tab)
		io.WriteString(s, frame.funcName)
		io.WriteString(s, "(...)")
		written = true
	}
}

func writeCallsite(w io.Writer, f *frame) {
	w.Write([]byte(f.file))
	w.Write([]byte(":"))
	w.Write(strconv.AppendInt(nil, int64(f.line), 10))
}

func unwrap(e error) iter.Seq[error] {
	return func(yield func(error) bool) {
		for {
			if e == nil || !yield(e) {
				return
			}
			e = errors.Unwrap(e)
		}
	}
}
