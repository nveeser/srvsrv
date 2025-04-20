package testdata

import (
	"embed"
	"testing"
)

//go:embed *.txt
var f embed.FS

func Read(t *testing.T, name string) string {
	data, err := f.ReadFile(name)
	if err != nil {
		t.Errorf("Error loading file: %s: %v", name, err)
	}
	return string(data)
}
