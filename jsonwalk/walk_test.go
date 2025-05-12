package jsonwalk

import (
	"github.com/google/go-cmp/cmp/cmpopts"
	"slices"
	"testing"
)

func TestWalk(t *testing.T) {
	tests := []struct {
		name       string
		visitor    *collect
		wantPaths  []string
		wantResult Result
	}{
		{
			name:    "all",
			visitor: &collect{},
			wantPaths: []string{
				"$.a",
				"$.a.value",
				"$.a.seq",
				"$.a.seq.0",
				"$.a.seq.0.obj1",
				"$.a.seq.0.obj1.name",
				"$.a.seq.1",
				"$.a.seq.1.obj2",
				"$.a.seq.1.obj2.name",
				"$.a.seq.2",
				"$.a.seq.2.obj3",
				"$.a.seq.2.obj3.name",
			},
		},
		{
			name: "skip/sequence-all",
			visitor: &collect{
				results: map[string]Result{
					"$.a.seq": Skip,
				},
			},
			wantPaths: []string{
				"$.a",
				"$.a.value",
				"$.a.seq",
			},
		},
		{
			name: "skip/sequence-element",
			visitor: &collect{
				results: map[string]Result{
					"$.a.seq.0": Skip,
				},
			},
			wantPaths: []string{
				"$.a",
				"$.a.value",
				"$.a.seq",
				"$.a.seq.0",
				"$.a.seq.1",
				"$.a.seq.1.obj2",
				"$.a.seq.1.obj2.name",
				"$.a.seq.2",
				"$.a.seq.2.obj3",
				"$.a.seq.2.obj3.name",
			},
		},
		{
			name: "exit",
			visitor: &collect{
				results: map[string]Result{
					"$.a.seq.0": Exit,
				},
			},
			wantResult: Exit,
			wantPaths: []string{
				"$.a",
				// May be present depending on the order that keys are called in.
				// "$.a.value",
				"$.a.seq",
				"$.a.seq.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := readJSON(t, "testdata/walk.json")
			r := Walk(obj, tt.visitor)
			if r != tt.wantResult {
				t.Errorf("Walk() = %v, want %v", r, tt.wantResult)
			}
			got := tt.visitor.paths

			var missing []string
			for _, required := range tt.wantPaths {
				if !slices.Contains(got, required) {
					missing = append(missing, required)
				}
			}
			if len(missing) > 0 {
				t.Errorf("Walk() did not see the following paths: %s", missing)
				t.Log(tt.visitor.paths)
				t.Log(tt.wantPaths)
			}
		})
	}
}

type collect struct {
	paths   []string
	results map[string]Result
}

func (c *collect) Mapping(p Path, _ Mapping, v map[string]any) Result { return c.collect(p) }
func (c *collect) Sequence(p Path, _ Mapping, v []any) Result         { return c.collect(p) }
func (c *collect) Scalar(p Path, _ Mapping, v any) Result             { return c.collect(p) }

func (c *collect) collect(path Path) Result {
	c.paths = append(c.paths, path.String())
	if c.results != nil {
		if r, ok := c.results[path.String()]; ok {
			return r
		}
	}
	return Continue
}

var sortSlicesOpt = cmpopts.SortSlices(func(a, b string) bool { return a < b })
