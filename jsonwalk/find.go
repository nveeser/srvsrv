package jsonwalk

import (
	"iter"
	"slices"
	"strings"
)

func Find(obj any, path string) iter.Seq[any] {
	return func(yield func(any) bool) {
		v := &findVisitor{
			path:  ParsePath(path),
			yield: yield,
		}
		Walk(obj, v)
	}
}

func FindOne(obj any, path string) (any, bool) {
	if strings.Contains(path, "*") {
		panic("path may contain multiple values: " + path)
	}
	found := slices.Collect(Find(obj, path))
	switch len(found) {
	case 1:
		return found[0], true
	case 0:
		return nil, false
	default:
		panic("path contains multiple values: " + path)
	}
}

type findVisitor struct {
	path  Path
	yield func(any) bool
}

func (f *findVisitor) Object(p Path, v map[string]any) Result { return f.check(p, v) }
func (f *findVisitor) Sequence(p Path, v []any) Result        { return f.check(p, v) }
func (f *findVisitor) Scalar(p Path, v any) Result            { return f.check(p, v) }

func (f *findVisitor) check(p Path, v any) Result {
	if !p.Match(f.path) {
		return Skip
	}
	if p.Len() == f.path.Len() {
		if !f.yield(v) {
			return Exit
		}
	}
	return Continue
}
