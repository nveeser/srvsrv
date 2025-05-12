package jsonwalk

import (
	"reflect"
	"slices"
	"testing"
)

func TestFind(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantCount int
		wantType  any
	}{
		{name: "obj", path: "$.a", wantCount: 1, wantType: map[string]any{}},
		{name: "scalar", path: "$.a.value", wantCount: 1, wantType: "string"},
		{name: "sequence", path: "$.a.seq", wantCount: 1, wantType: []any{nil}},
		{name: "repeated/all", path: "$.a.seq.*.obj.name", wantCount: 3, wantType: "string"},
		{name: "repeated/single", path: "$.a.seq.1.obj.name", wantCount: 1, wantType: "string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := readJSON(t, "testdata/find.json")
			got := slices.Collect(Find(obj, tt.path))
			if len(got) > 0 && tt.wantType != nil {
				if reflect.TypeOf(got[0]) != reflect.TypeOf(tt.wantType) {
					t.Errorf("got type %T wanted %T", got[0], tt.wantType)
				}
			}
			if len(got) != tt.wantCount {
				t.Errorf("got wantCount %d wanted %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestFindOne(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantFound bool
		wantType  any
	}{
		{name: "obj", path: "$.a", wantFound: true, wantType: map[string]any{}},
		{name: "scalar", path: "$.a.value", wantFound: true, wantType: "string"},
		{name: "sequence", path: "$.a.seq", wantFound: true, wantType: []any{nil}},
		{name: "repeated", path: "$.a.seq.2", wantFound: true},
		{name: "empty", path: "$.a.no.path", wantFound: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := readJSON(t, "testdata/find.json")
			got, found := FindOne(obj, tt.path)
			if found != tt.wantFound {
				t.Errorf("FindOne() got found %t wanted %t", found, tt.wantFound)
			}
			if tt.wantType != nil {
				if reflect.TypeOf(got) != reflect.TypeOf(tt.wantType) {
					t.Errorf("got type %T wanted %T", got, tt.wantType)
				}
			}
		})
	}
}
