package jsonwalk

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Printer struct{}

func (p Printer) Object(path Path, v map[string]any) Result {
	fmt.Printf("%s:object [%v]\n", indent(path), slices.Collect(maps.Keys(v)))
	return Continue
}

func (p Printer) Sequence(path Path, v []any) Result {
	fmt.Printf("%s:sequence(size=%d)\n", indent(path), len(v))
	return Continue
}

func (p Printer) Scalar(path Path, v any) Result {
	vs := fmt.Sprintf("%v", v)
	if len(vs) > 100 {
		vs = vs[:100]
	}
	fmt.Printf("%s:scalar %s\n", indent(path), vs)
	return Continue
}

func indent(p Path) string {
	return fmt.Sprintf("%s[%s]", strings.Repeat(" ", p.Len()), p)
}
