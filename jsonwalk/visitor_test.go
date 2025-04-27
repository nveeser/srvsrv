package jsonwalk

import "testing"

func TestPath(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		match string
		want  bool
	}{
		{
			name:  "exact/match",
			path:  "k1.k2.k3",
			match: "k1.k2.k3",
			want:  true,
		},
		{
			name:  "exact/no-match",
			path:  "k1.k2.k3",
			match: "k1.x.k3",
			want:  false,
		},
		{
			name:  "short/match",
			path:  "k1.k2",
			match: "k1.k2.k3",
			want:  true,
		},
		{
			name:  "short/no-match",
			path:  "k1.k2",
			match: "k1.x.k3",
			want:  false,
		},
		{
			name:  "long/match",
			path:  "k1.k2.k3",
			match: "k1.k2",
			want:  true,
		},
		{
			name:  "long/no-match",
			path:  "k1.k2.k3",
			match: "k1.x",
			want:  false,
		},
		{
			name:  "numeric/match",
			path:  "k1.3.k3",
			match: "k1.*.k3",
			want:  true,
		},
		{
			name:  "numeric/no-match",
			path:  "k1.k2.k3",
			match: "k1.*.k3",
			want:  false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := ParsePath(tc.path)
			m := ParsePath(tc.match)
			got := p.Match(m)
			if got != tc.want {
				t.Errorf("Matches got %t wantPaths %t", got, tc.want)
			}
		})

	}
}
