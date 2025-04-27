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

//type Node struct {
//	Path   Path
//	Object map[string]any
//	Key    string
//}

type Visitor interface {
	Object(p Path, v map[string]any) Result
	Sequence(p Path, v []any) Result
	Scalar(p Path, v any) Result
}

type Path []string

func ParsePath(s string) Path {
	return strings.Split(s, ".")
}
func (p Path) String() string { return strings.Join(p, ".") }
func (p Path) Len() int       { return len(p) }

func (p Path) Key() string         { return p[len(p)-1] }
func (p Path) Parent() Path        { return p[0 : len(p)-1] }
func (p Path) Child(s string) Path { return append(p, s) }

func (p Path) Equal(o Path) bool    { return len(p) == len(o) && p.Match(o) }
func (p Path) Prefix(o Path) bool   { return len(p) <= len(o) && p.Match(o) }
func (p Path) Contains(o Path) bool { return len(p) >= len(o) && p.Match(o) }

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
