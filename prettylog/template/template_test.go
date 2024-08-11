package template

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestTemplateFuncs(t *testing.T) {
	cases := []struct {
		name   string
		format string
		data   map[string]string
		want   string
	}{
		{
			name:   "pre/non-empty",
			format: `{.time}{.level | pre ":" }{.msg | pre " "}`,
			data: map[string]string{
				"time":  "2006-12-12",
				"level": "INFO",
				"msg":   "Message",
			},
			want: "2006-12-12:INFO Message",
		},
		{
			name:   "pre/empty",
			format: `{.time}{.level | pre ":" }{.msg | pre " "}`,
			data: map[string]string{
				"time":  "2006-12-12",
				"level": "",
				"msg":   "Message",
			},
			want: "2006-12-12 Message",
		},
		{
			name:   "preln/non-empty",
			format: `[{.time}] {.level}{.msg | preln}`,
			data: map[string]string{
				"time":  "2006-12-12",
				"level": "INFO",
				"msg":   "Message",
			},
			want: "[2006-12-12] INFO\nMessage",
		},
		{
			name:   "preln/empty",
			format: `[{.time}] {.level}{.msg | preln}`,
			data: map[string]string{
				"time":  "2006-12-12",
				"level": "INFO",
				"msg":   "Message",
			},
			want: "[2006-12-12] INFO\nMessage",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := Parse(tc.format)
			if err != nil {
				t.Fatalf("error parsing template: %s", err)
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, tc.data); err != nil {
				t.Errorf("Execute got an error: %s", err)
			}
			got := buf.String()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Got diff -want +got: %s", diff)
				t.Logf("GOT: %q", got)
			}
		})
	}
}

func TestTemplateKeys(t *testing.T) {
	cases := []struct {
		name     string
		format   string
		wantKeys []string
	}{
		{
			name:     "basic",
			format:   `{.time}{.level | pre "" }: {.msg}`,
			wantKeys: []string{"time", "level", "msg"},
		},
		{
			name:   "json",
			format: "{.time} {.level}: {.msg}{.json}",
			wantKeys: []string{
				"time", "level", "msg", "json",
			},
		},
		{
			name:   "json",
			format: `{.time} {.level}: {.msg}{.json | pre "\n" }`,
			wantKeys: []string{
				"time", "level", "msg", "json",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := Parse(tc.format)
			if err != nil {
				t.Fatalf("error parsing template: %s", err)
			}
			keys := tmpl.Keys()
			if err != nil {
				t.Errorf("parse got error: %s", err)
			}
			if diff := cmp.Diff(tc.wantKeys, keys); diff != "" {
				t.Errorf("Got diff -want +got: %s", diff)
			}
		})
	}
}
