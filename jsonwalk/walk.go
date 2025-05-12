package jsonwalk

import (
	"encoding/json"
	"iter"
	"maps"
	"strconv"
)

func WalkJSON(d []byte, visit Visitor) (Result, error) {
	root := make(map[string]any)
	if err := json.Unmarshal(d, &root); err != nil {
		return Exit, err
	}
	return Walk(root, visit), nil
}

func Walk(obj Mapping, visit Visitor) Result {
	return walkIter(rootPath, obj, maps.All(obj), visit)
}

func walkObj(p Path, m Mapping, value any, visit Visitor) Result {
	switch v := value.(type) {
	case map[string]any:
		r := visit.Mapping(p, m, v)
		if r != Continue {
			return r
		}
		return walkIter(p, v, maps.All(v), visit)

	case []any:
		r := visit.Sequence(p, m, v)
		if r != Continue {
			return r
		}
		return walkIter(p, m, keyedSlice(v), visit)

	default:
		r := visit.Scalar(p, m, value)
		if r == Exit {
			return r
		}
	}
	return Continue
}

func walkIter(p Path, m Mapping, seq iter.Seq2[string, any], visit Visitor) Result {
	for k, v := range seq {
		r := walkObj(p.Child(k), m, v, visit)
		if r == Exit {
			return r
		}
	}
	return Continue
}

func keyedSlice(s []any) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for i, sv := range s {
			if !yield(strconv.Itoa(i), sv) {
				return
			}
		}
	}
}
