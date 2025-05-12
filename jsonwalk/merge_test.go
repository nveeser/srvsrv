package jsonwalk

import (
	"encoding/json"
	diffcmp "github.com/google/go-cmp/cmp"
	"testing"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name      string
		opts      []MergeOption
		dst       string
		src       string
		want      string
		wantPaths []string
	}{
		{
			name: "default",
			want: "testdata/merge-default.json",
		},
		{
			name: "ignore-scalar",
			want: "testdata/merge-ignore-scalar.json",
			opts: []MergeOption{
				IgnorePath("$.a.value"),
			},
		},
		{
			name: "ignore-sequence",
			want: "testdata/merge-ignore-sequence.json",
			opts: []MergeOption{
				IgnorePath("$.a.seq"),
			},
		},
		{
			name: "ignore-mapping",
			want: "testdata/merge-ignore-mapping.json",
			opts: []MergeOption{
				IgnorePath("$.b"),
			},
		},
		{
			name: "replace-sequence",
			want: "testdata/merge-replace-sequence.json",
			opts: []MergeOption{
				Replace("$.a.seq"),
			},
		},
		{
			name: "replace-mapping",
			want: "testdata/merge-replace-mapping.json",
			opts: []MergeOption{
				Replace("$.a.seq"),
			},
		},
	}
	type data struct {
		Dst  map[string]any
		Src  map[string]any
		Want map[string]any
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := read[data](t, tt.want)
			if input.Dst == nil {
				t.Fatalf("input is malformed")
			}
			err := Merge(input.Dst, input.Src, tt.opts...)
			if err != nil {
				t.Errorf("Merge got error: %s", err)
			}

			if diff := diffcmp.Diff(input.Want, input.Dst); diff != "" {
				t.Errorf("request got diff: -want/+got: %s", diff)
			}
		})
	}
}

func mustUnmarshal(t *testing.T, d []byte) map[string]any {
	t.Helper()
	m := make(map[string]any)
	if err := json.Unmarshal(d, &m); err != nil {
		t.Fatalf("json.Unmarshal() got err: %s", err)
	}
	return m
}
