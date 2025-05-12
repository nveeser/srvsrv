package jsonwalk

import (
	"encoding/json"
	"os"
	"testing"
)

func readJSON(t *testing.T, filename string) map[string]any {
	d, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("error reading test file %s: %v", filename, err)
	}
	obj := make(map[string]any)
	if err = json.Unmarshal(d, &obj); err != nil {
		t.Fatalf("error unmarshing: %s: %v", filename, err)
	}
	return obj
}

func read[T any](t *testing.T, filename string) T {
	d, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("error reading test file %s: %v", filename, err)
	}
	var value T
	if err = json.Unmarshal(d, &value); err != nil {
		t.Fatalf("error unmarshing: %s: %v", filename, err)
	}
	return value
}
