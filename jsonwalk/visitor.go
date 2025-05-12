package jsonwalk

import (
	"strconv"
	"strings"
)

type Result int

const (
	Continue Result = iota
	Skip
	Exit
)

type Mapping = map[string]any

type Visitor interface {
	Mapping(p Path, m Mapping, v map[string]any) Result
	Sequence(p Path, m Mapping, v []any) Result
	Scalar(p Path, m Mapping, v any) Result
}

type EmptyVisitor struct{}

func (f EmptyVisitor) Mapping(p Path, m Mapping, v map[string]any) Result { return Continue }
func (f EmptyVisitor) Sequence(p Path, m Mapping, v []any) Result         { return Continue }
func (f EmptyVisitor) Scalar(p Path, m Mapping, v any) Result             { return Continue }

type NodeVisitor func(p Path, m Mapping, v any) Result

func (f NodeVisitor) Mapping(p Path, m Mapping, v map[string]any) Result { return f(p, m, v) }
func (f NodeVisitor) Sequence(p Path, m Mapping, v []any) Result         { return f(p, m, v) }
func (f NodeVisitor) Scalar(p Path, m Mapping, v any) Result             { return f(p, m, v) }

type MappingVisitor func(p Path, m Mapping, v map[string]any) Result

func (f MappingVisitor) Mapping(p Path, m Mapping, v map[string]any) Result { return f(p, m, v) }
func (f MappingVisitor) Sequence(p Path, m Mapping, v []any) Result         { return Continue }
func (f MappingVisitor) Scalar(p Path, m Mapping, v any) Result             { return Continue }

type SequenceVisitor func(p Path, m Mapping, v []any) Result

func (f SequenceVisitor) Mapping(p Path, m Mapping, v map[string]any) Result { return Continue }
func (f SequenceVisitor) Sequence(p Path, m Mapping, v []any) Result         { return f(p, m, v) }
func (f SequenceVisitor) Scalar(p Path, m Mapping, v any) Result             { return Continue }

type ScalarVisitor func(p Path, m Mapping, v any) Result

func (f ScalarVisitor) Mapping(p Path, m Mapping, v map[string]any) Result { return Continue }
func (f ScalarVisitor) Sequence(p Path, m Mapping, v []any) Result         { return Continue }
func (f ScalarVisitor) Scalar(p Path, m Mapping, v any) Result             { return f(p, m, v) }

type Path []string

var rootPath Path = []string{"$"}

func ParsePath(s string) Path {
	return strings.Split(s, ".")
}
func (p Path) String() string { return strings.Join(p, ".") }
func (p Path) Len() int       { return len(p) }
func (p Path) Relative() bool { return len(p) > 0 && p[0] != "$" }
func (p Path) IsRoot() bool   { return len(p) == 1 && p[0] == "$" }

func (p Path) Key() string         { return p[len(p)-1] }
func (p Path) Parent() Path        { return p[0 : len(p)-1] }
func (p Path) Child(s string) Path { return append(p, s) }
func (p Path) at(i int) string     { return p[i] }

func (p Path) Equal(o Path) bool  { return len(p) == len(o) && p.Match(o) }
func (p Path) Prefix(o Path) bool { return len(p) <= len(o) && p.Match(o) }

func (p Path) Match(o Path) bool {
	for i := 0; i < len(p); i++ {
		if len(o) <= i {
			return true
		}
		if o[i] == "*" && isNumber(p[i]) {
			return true
		}
		if p[i] != o[i] {
			return false
		}
	}
	return true
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
