package ctxerr

import (
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/nveeser/srvsrv/ctxerr/testdata"
	"runtime"
	"strings"
	"testing"
)

var stackPathPrefix string

func init() {
	stackPathPrefix = "/no/prefix/found"
	if _, file, _, ok := runtime.Caller(0); ok {
		i := strings.Index(file, "ctxerr")
		stackPathPrefix = file[:i]
	}
}

func setupFrame() func() {
	orig := newFrame
	newFrame = func(i int, f runtime.Frame) *frame {
		frame := orig(i, f)
		if strings.HasPrefix(frame.file, stackPathPrefix) {
			frame.file = strings.Replace(frame.file, stackPathPrefix, "/foo/src/", 1)
			frame.line = i + 10*10
		}
		return frame
	}
	return func() {
		newFrame = orig
	}
}

// fmt.Formatter %s
// fmt.Formatter %v
// fmt.Formatter %+v
// Op / Msg / Cause
// Cause { nil, Error, error }

func TestFormatString(t *testing.T) {
	cases := []struct {
		name  string
		input error
		want  string
	}{
		{
			name:  "Op",
			input: E(Op("open-db")),
			want:  "[open-db]",
		},
		{
			name:  "Msg",
			input: E("Message for new error"),
			want:  "Message for new error",
		},
		{
			name:  "Op/Msg",
			input: E(Op("open-db"), "Message for new error"),
			want:  "[open-db]: Message for new error",
		},
		{
			name:  "Op/Cause",
			input: E(Op("open-db"), errors.New("error opening db")),
			want:  "[open-db]: error opening db",
		},
		{
			name:  "Msg/Cause",
			input: E("Message for new error", errors.New("error opening db")),
			want:  "Message for new error: error opening db",
		},
		{
			name:  "Msg/Op/cause",
			input: E(Op("open-db"), "Message for new error", errors.New("error opening db")),
			want:  "[open-db]: Message for new error: error opening db",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fmt.Sprintf("%s", tc.input)
			if diff := cmp.Diff(tc.want, got, cmpopts.AcyclicTransformer("trim", trimWhitespace)); diff != "" {
				t.Errorf("Diff: -want/+got %s", diff)
				t.Logf("got\n%s\n", got)
				t.Logf("wanted\n%s\n", tc.want)
			}
		})
	}
}

func TestFormatVerbose(t *testing.T) {
	err := myFunc1(errors.New("connection error"))
	err = Ef(Op("open-db"), err, "opening database: %s", "addr=172.10.10.10:256")
	err = E(Op("init-storage"), "error reading", err)
	err = fmt.Errorf("error: starting database: %w", err)
	err = E(Op("start-server"), err)

	cases := []struct {
		name     string
		format   string
		wantFile string
	}{
		{
			name:     "chain/no-stack",
			format:   "%x",
			wantFile: "chain.txt",
		},
		{
			name:     "chain/no-stack/indent",
			format:   "%3x",
			wantFile: "chain-indent.txt",
		},
		{
			name:     "chain/stack",
			format:   "%+x",
			wantFile: "chain-stack.txt",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			done := setupFrame()
			defer done()
			got := fmt.Sprintf(tc.format, err)
			want := testdata.Read(t, tc.wantFile)

			if diff := cmp.Diff(want, got, cmpopts.AcyclicTransformer("trim", trimWhitespace)); diff != "" {
				t.Errorf("Diff: -want/+got %s", diff)
				fmt.Printf("got\n%s\n", got)
				fmt.Printf("wanted\n%s\n", want)
			}
		})
	}
	if t.Failed() {
		dumpStacks(t, err)
	}
}

func dumpStacks(t *testing.T, err error) {
	t.Logf("-----[Cause Chain]----")
	var last *Error
	for e := range unwrap(err) {
		t.Logf("{Cause} %s", e)
		if x, ok := e.(*Error); ok {
			for i, frame := range x.stack.frames() {
				t.Logf("  [%d] %+v", i, frame)
			}
			last = x
		}
	}

	t.Logf("-----[Caller Stack]----")
	for i, frame := range callers(1).frames() {
		t.Logf("[%d] %+v", i, frame)
	}
	t.Logf("______[Walk Stack]______")
	for f := range walkStack(last.stack, 1) {
		t.Logf("%+v", f)
	}
}

func trimWhitespace(s string) string {
	s = strings.TrimSpace(s)
	//re := regexp.MustCompile(`\s+`)
	//s = re.ReplaceAllString(s, " ")
	return s
}
