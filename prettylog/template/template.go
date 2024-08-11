// Package template provides a Template wrapper that allows extracting the
// identifiers present in the template. Primarily used by prettylog to partition
// attributes into the template and add the remaining.
package template

import (
	"fmt"
	"text/template"
	"text/template/parse"
)

type Option func(*template.Template) *template.Template

// Delims sets the delimiter for the template during parsing.
func Delims(l, r string) Option {
	return func(t *template.Template) *template.Template {
		return t.Delims(l, r)
	}
}

// Parse parse the specified format into a template and extracts
// the identifiers in the actions present in the template.
func Parse(format string, opts ...Option) (*KeyedTemplate, error) {
	tmpl := template.New("log").Delims("{", "}")
	tmpl.Funcs(functionMap)
	for _, o := range opts {
		tmpl = o(tmpl)
	}
	tmpl, err := tmpl.Parse(format)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}
	var out []string
	walkTree(0, tmpl.Tree.Root, func(keys []string, depth int) {
		if len(keys) > 0 {
			out = append(out, keys[0])
		}
	})

	return &KeyedTemplate{
		Template: tmpl,
		keys:     out,
	}, nil
}

// KeyedTemplate wraps a text/template.Template instance and contains the list of
// identifiers (aka keys) that are referenced by the format.
type KeyedTemplate struct {
	*template.Template
	keys []string
}

func (t *KeyedTemplate) Keys() []string {
	return t.keys
}

var functionMap = map[string]any{
	"pre": func(arg, value string) string {
		if value == "" {
			return ""
		}
		return arg + value
	},

	"post": func(arg, value string) string {
		if value == "" {
			return ""
		}
		return value + arg
	},
	"preln": func(value string) string {
		if value == "" {
			return ""
		}
		return "\n" + value
	},
	"postln": func(value string) string {
		if value == "" {
			return ""
		}
		return value + "\n"
	},
	"pres": func(value string) string {
		if value == "" {
			return ""
		}
		return " " + value
	},
	"posts": func(value string) string {
		if value == "" {
			return ""
		}
		return value + " "
	},
}

func walkTree(d int, node parse.Node, fn func(s []string, depth int)) {
	switch n := node.(type) {
	case *parse.ListNode:
		for _, n := range n.Nodes {
			walkTree(d+1, n, fn)
		}
	case *parse.ActionNode:
		walkTree(d+1, n.Pipe, fn)
	case *parse.PipeNode:
		for _, x := range n.Cmds {
			walkTree(d+1, x, fn)
		}
	case *parse.CommandNode:
		for _, x := range n.Args {
			walkTree(d+1, x, fn)
		}
	case *parse.FieldNode:
		fn(n.Ident, d)
	}
}
