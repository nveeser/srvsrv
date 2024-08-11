package ctxerr

import (
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"runtime"
	"strings"
	"testing"
)

var stackPathPrefix string

func init() {
	stackPathPrefix = "/no/prefix/found"
	_, file, _, ok := runtime.Caller(0)
	if ok {
		i := strings.Index(file, "ctxerr")
		stackPathPrefix = file[:i]
	}
}

func setupFrame() func() {
	orig := callerFrame
	var count int
	callerFrame = func(p []uintptr, n int) *frame {
		count++
		frame := orig(p, n)
		if strings.HasPrefix(frame.file, stackPathPrefix) {
			frame.file = strings.Replace(frame.file, stackPathPrefix, "/foo/src/", 1)
			frame.line = count
		}
		return frame
	}
	return func() {
		callerFrame = orig
	}
}

func TestFormatError(t *testing.T) {
	done := setupFrame()
	defer done()

	t.Run("Functions", func(t *testing.T) {
		err := myFunc1()
		got := fmt.Sprintf("%+v", err)
		var want = `
[op] 
error happened
	/foo/src/ctxerr/errors_test.go:7 
	   srvsrv/ctxerr.myFunc1(...)
	/foo/src/ctxerr/errors_test.go:9 
	   srvsrv/ctxerr.T.myFunc2(...)
	/foo/src/ctxerr/errors_test.go:10 
	   srvsrv/ctxerr.myFunc3(...)`

		if diff := cmp.Diff(want, got, cmpopts.AcyclicTransformer("trim", strings.TrimSpace)); diff != "" {
			t.Logf("Diff: -want/+got %s", diff)
			t.Logf("got\n%s\n", got)
			t.Logf("wanted\n%s\n", want)
			t.Fail()
		}
	})
	t.Run("Wrapping", func(t *testing.T) {
		err := E(Op("one"), "error", E(Op("two"), "error building foo", E(Op("three"), "error building bar", errors.New("concrete"))))
		got := fmt.Sprintf("%+v", err)
		want := `
[one] : error
[two] : error building foo
[three] : error building bar
concrete
`
		if diff := cmp.Diff(want, got, cmpopts.AcyclicTransformer("trim", strings.TrimSpace)); diff != "" {
			t.Logf("Diff: -want/+got %s", diff)
			t.Logf("got\n%s\n", got)
			t.Logf("wanted\n%s\n", want)
			t.Fail()
		}
	})
}

//go:noinline
func myFunc1() error {
	var t T
	return t.myFunc2()
}

type T struct{}

//go:noinline
func (T) myFunc2() error {
	return myFunc3()
}

//go:noinline
func myFunc3() error {
	return E(Op("op"), myFunc4())
}

//go:noinline
func myFunc4() error {
	return errors.New("error happened")
}
