package jsonwalk

import (
	"encoding/json"
	"fmt"
	"strings"
)

var Debug bool = false

func debug(p Path, event string, _ any) {
	fmt.Printf("%s[%s]: %s\n", strings.Repeat(" ", p.Len()), p, event)
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

// DebugFunc specifies a debug function for the given path.
func DebugFunc(f func(p Path, event string, value any)) MergeOption {
	return func(v *mergeVisitor) {
		if f == nil {
			v.debug = nil
		} else {
			v.debug = f
		}
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
		debug:     func(p Path, event string, value any) {},
	}
	if Debug {
		v.debug = debug
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
	debug     func(p Path, event string, value any)
}

func (m *mergeVisitor) Object(p Path, v object) Result {
	if p == nil {
		m.debug(p, "object skip root", v)
		return Continue
	}
	parent, dstObj, err := parentValue[object](m.dstByPath, p)
	if err != nil {
		m.error = err
		return Exit
	}
	switch {
	case m.opts.strategy(p) == pathIgnore:
		m.debug(p, "object ignore", v)
		dstObj = v
	case dstObj == nil:
		m.debug(p, "object replace", v)
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
		m.debug(p, "sequence ignore", v)
		return Skip
	case strat == pathReplace:
		m.debug(p, "sequence replace", v)
		dstSeq = v
	case dstSeq == nil:
		m.debug(p, "sequence add", v)
		dstSeq = v
	default:
		m.debug(p, "slice append", v)
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
		m.debug(p, "scalar ignore", v)
	case dstAny == nil:
		m.debug(p, "scalar set", v)
		dstAny = v
	default:
		m.debug(p, "scalar replace", v)
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
