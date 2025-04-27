package jsonwalk

import (
	"encoding/json"
	"fmt"
	"iter"
	"slices"
	"strings"
)

func FindJSON(data []byte, path string) ([][]byte, error) {
	obj := make(map[string]any)
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("error Unmarshal data: %w", err)
	}
	var out [][]byte
	for obj := range Find(obj, path) {
		jsonOut, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("error Marshal() %w", err)
		}
		out = append(out, jsonOut)
	}
	return out, nil
}

func FindOneJSON(data []byte, path string) ([]byte, bool, error) {
	obj := make(map[string]any)
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, false, fmt.Errorf("error Unmarshal data: %w", err)
	}
	foundObj, ok := FindOne(obj, path)
	if !ok {
		return nil, false, nil
	}
	foundJSON, err := json.Marshal(foundObj)
	if err != nil {
		return nil, false, fmt.Errorf("error Marshal() %w", err)
	}
	return foundJSON, true, err
}

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
