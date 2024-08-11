package ctxerr

import (
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"strings"
	"testing"
)

func init() {
	fakeLineNumbers = true
}

func TestFormatError(t *testing.T) {
	t.Run("Functions", func(t *testing.T) {
		err := func1()
		got := fmt.Sprintf("%+v", err)
		var want = `
[op] 
error happened
	/home/nicholas/radar/ctxerr/errors_test.go:0 
	   radar/ctxerr.func1(...)
	/home/nicholas/radar/ctxerr/errors_test.go:0 
	   radar/ctxerr.T.func2(...)
`

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

func func1() error {
	var t T
	return t.func2()
}

type T struct{}

func (T) func2() error {
	return E(Op("op"), func3())
}

func func3() error {
	return func4()
}

func func4() error {
	return errors.New("error happened")
}
