package jsonwalk

import (
	"encoding/json"
	"fmt"
	"strings"
)

var Debug bool = false

func debugf(p Path, format string, args ...any) {
	if Debug {
		indent := strings.Repeat(" ", p.Len())
		format = "%s[%s]" + format
		extra := []any{indent, p}
		args = append(extra, args...)
		fmt.Printf(format, args...)
	}
}

type MergeOption func(visitor *mergeVisitor)

func IgnorePath(path string) MergeOption {
	return func(v *mergeVisitor) {
		v.opts.put(ParsePath(path), pathIgnore)
	}
}

func Replace(path string) MergeOption {
	return func(v *mergeVisitor) {
		v.opts.put(ParsePath(path), pathReplace)
	}
}

func Append(path string) MergeOption {
	return func(v *mergeVisitor) {
		v.opts.put(ParsePath(path), pathAppend)
	}
}

type object = map[string]any

func MergeJSON(dstJSON, srcJSON []byte, opts ...MergeOption) ([]byte, error) {
	dstObj := make(map[string]any)
	if err := json.Unmarshal(dstJSON, &dstObj); err != nil {
		return nil, fmt.Errorf("error Unmarshal dstJSON: %w", err)
	}
	srcObj := make(map[string]any)
	if err := json.Unmarshal(srcJSON, &srcObj); err != nil {
		return nil, fmt.Errorf("error Unmarshal srcObj: %w", err)
	}
	if err := Merge(dstObj, dstObj, opts...); err != nil {
		return nil, err
	}
	d, err := json.Marshal(dstObj)
	if err != nil {
		return nil, fmt.Errorf("error Marshal merged obj: %w", err)
	}
	return d, nil
}

func Merge(dst, src map[string]any, opts ...MergeOption) error {
	v := &mergeVisitor{
		dst:       dst,
		dstByPath: make(map[string]object),
	}
	v.dstByPath[""] = dst
	for _, opt := range opts {
		opt(v)
	}
	Walk(src, v)
	return v.error
}

type mergeVisitor struct {
	dst  object
	opts optionTrie
	// objects by path
	dstByPath map[string]object
	error     error
}

func (m *mergeVisitor) Object(p Path, v object) Result {
	if p == nil {
		debugf(p, "%s:object skip root\n")
		return Continue
	}
	parent, dstObj, err := parentValue[object](m.dstByPath, p)
	if err != nil {
		m.error = err
		return Exit
	}
	switch {
	case dstObj == nil:
		debugf(p, "%s:object replace\n")
		dstObj = v
	case m.opts.strategy(p) == pathIgnore:
		debugf(p, "%s:object ignore\n")
		dstObj = v
	}
	m.dstByPath[p.String()] = dstObj
	parent[p.Key()] = dstObj
	return Continue
}

func (m *mergeVisitor) Sequence(p Path, v []any) Result {
	strat := m.opts.strategy(p)

	parent, dstSeq, err := parentValue[[]any](m.dstByPath, p)
	if err != nil {
		m.error = err
		return Exit
	}
	switch {
	case strat == pathIgnore:
		debugf(p, "%s:sequence ignore\n")
		return Skip
	case strat == pathReplace:
		debugf(p, "%s:sequence replace\n")
		dstSeq = v
	case dstSeq == nil:
		debugf(p, "%s:sequence add\n")
		dstSeq = v
	default:
		debugf(p, "%s:slice append\n")
		dstSeq = append(dstSeq, v...)
	}
	parent[p.Key()] = dstSeq
	return Skip
}

func (m *mergeVisitor) Scalar(p Path, v any) Result {
	parent, dstAny, err := parentValue[any](m.dstByPath, p)
	if err != nil {
		m.error = err
		return Exit
	}
	switch {
	case m.opts.strategy(p) == pathIgnore:
		debugf(p, "%s:scalar ignore\n")
	case dstAny == nil:
		debugf(p, "%s:scalar set\n")
		dstAny = v
	default:
		debugf(p, "%s:scalar replace\n")
		dstAny = v
	}
	parent[p.Key()] = dstAny
	return Continue
}

func parentValue[T any](dstCache map[string]object, path Path) (object, T, error) {
	var zero T
	parent, ok := dstCache[path.Parent().String()]
	if !ok {
		return nil, zero, fmt.Errorf("no parent object at %s", path.Parent())
	}
	dstAny, ok := parent[path.Key()]
	if !ok {
		return parent, zero, nil
	}
	dstValue, ok := dstAny.(T)
	if !ok {
		return parent, zero, fmt.Errorf("invalid type at %s: was %T expected %T", path, dstValue, zero)
	}
	return parent, dstValue, nil
}
