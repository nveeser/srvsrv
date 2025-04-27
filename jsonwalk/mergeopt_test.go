package jsonwalk

import "testing"

func TestOptionSet(t *testing.T) {
	var os optionTrie
	paths := []string{
		"a.b.1",
		"a.b.*.c",
		"a.b.2.c",
		"a.x.m",
		"a.y.n",
		"a.y.x",
	}
	for _, p := range paths {
		os.put(ParsePath(p), strategyAppend)
	}
	{
		got := os.strategy(ParsePath("a.b.1"))
		if got != strategyAppend {
			t.Errorf("strategy got %s wantPaths %s", got, strategyAppend)
		}
	}
	{
		got := os.strategy(ParsePath("a.b.1.c"))
		if got != strategyAppend {
			t.Errorf("strategy got %s wantPaths %s", got, strategyAppend)
		}
	}
}
