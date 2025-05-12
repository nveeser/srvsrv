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
		v.strategies.put(ParsePath(path), pathIgnore)
	}
}

func Replace(path string) MergeOption {
	return func(v *mergeVisitor) {
		v.strategies.put(ParsePath(path), pathReplace)
	}
}

func Append(path string) MergeOption {
	return func(v *mergeVisitor) {
		v.strategies.put(ParsePath(path), pathAppend)
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
	dstMap := make(map[string]any)
	if err := json.Unmarshal(dstJSON, &dstMap); err != nil {
		return nil, fmt.Errorf("error Unmarshal dstJSON: %w", err)
	}
	srcObj := make(map[string]any)
	if err := json.Unmarshal(srcJSON, &srcObj); err != nil {
		return nil, fmt.Errorf("error Unmarshal srcObj: %w", err)
	}
	if err := Merge(dstMap, dstMap, opts...); err != nil {
		return nil, err
	}
	d, err := json.Marshal(dstMap)
	if err != nil {
		return nil, fmt.Errorf("error Marshal merged obj: %w", err)
	}
	return d, nil
}

func Merge(dst, src map[string]any, opts ...MergeOption) error {
	v := &mergeVisitor{
		dst:        dst,
		strategies: &strategyTrie{},
		dstByPath:  make(map[string]object),
		debug:      func(p Path, event string, value any) {},
	}
	if Debug {
		v.debug = debug
	}
	v.dstByPath[rootPath.String()] = dst
	for _, opt := range opts {
		opt(v)
	}
	Walk(src, v)
	return v.error
}

type mergeVisitor struct {
	dst        object
	strategies *strategyTrie
	dstByPath  map[string]object // objects by path

	stackPath Path
	stack     []object

	error error
	debug func(p Path, event string, value any)
}

func (m *mergeVisitor) Mapping(p Path, _ Mapping, v object) Result {
	if p.IsRoot() {
		m.debug(p, "object skip root", v)
		return Continue
	}
	dstMap, dstValue, err := findDstValue[object](m.dstByPath, p)
	if err != nil {
		m.error = err
		return Exit
	}
	strat := m.strategies.strategy(p)
	switch {
	case strat == pathIgnore:
		m.debug(p, "object ignore", v)
		return Skip
	case strat == pathReplace:
		m.debug(p, "object replace", v)
		dstMap[p.Key()] = v
		return Skip
	case dstValue == nil:
		m.debug(p, "object set-new", v)
		dstMap[p.Key()] = v
		return Skip
	default:
		m.dstByPath[p.String()] = dstValue
		return Continue
	}
}

func (m *mergeVisitor) Sequence(p Path, _ Mapping, v []any) Result {
	dstMap, dstSeq, err := findDstValue[[]any](m.dstByPath, p)
	if err != nil {
		m.error = err
		return Exit
	}
	strat := m.strategies.strategy(p)
	switch {
	case strat == pathIgnore:
		m.debug(p, "sequence ignore", v)
		return Skip
	case strat == pathReplace:
		m.debug(p, "sequence replace", v)
		dstMap[p.Key()] = v
		return Skip
	case dstSeq == nil:
		m.debug(p, "sequence add", v)
		dstMap[p.Key()] = v
		return Skip
	default:
		m.debug(p, "slice append", v)
		dstMap[p.Key()] = append(dstSeq, v...)
		return Skip
	}
}

func (m *mergeVisitor) Scalar(p Path, _ Mapping, v any) Result {
	dstMap, dstScalar, err := findDstValue[any](m.dstByPath, p)
	if err != nil {
		m.error = err
		return Exit
	}
	switch {
	case m.strategies.strategy(p) == pathIgnore:
		m.debug(p, "scalar ignore", v)
	case dstScalar == nil:
		m.debug(p, "scalar set", v)
		dstScalar = v
	default:
		m.debug(p, "scalar replace", v)
		dstScalar = v
	}
	dstMap[p.Key()] = dstScalar
	return Continue
}

func findDstValue[T any](dstCache map[string]object, path Path) (object, T, error) {
	var zero T
	dstMap, ok := dstCache[path.Parent().String()]
	if !ok {
		return nil, zero, fmt.Errorf("dest has no object at %s", path.Parent())
	}
	dstAny, ok := dstMap[path.Key()]
	if !ok {
		return dstMap, zero, nil
	}
	dstValue, ok := dstAny.(T)
	if !ok {
		return dstMap, zero, fmt.Errorf("invalid type at %s: was %T expected %T", path, dstValue, zero)
	}
	return dstMap, dstValue, nil
}
