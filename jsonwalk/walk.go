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
	return walkObj(nil, root, visit), nil
}

func Walk(obj any, visit Visitor) Result {
	return walkObj(nil, obj, visit)
}

func walkObj(p Path, value any, visit Visitor) Result {
	switch v := value.(type) {
	case map[string]any:
		r := visit.Object(p, v)
		if r != Continue {
			return r
		}
		return walkIter(p, maps.All(v), visit)

	case []any:
		r := visit.Sequence(p, v)
		if r != Continue {
			return r
		}
		return walkIter(p, keyedSlice(v), visit)

	default:
		r := visit.Scalar(p, value)
		if r == Exit {
			return r
		}
	}
	return Continue
}

func walkIter(p Path, seq iter.Seq2[string, any], visit Visitor) Result {
	for k, v := range seq {
		r := walkObj(p.Child(k), v, visit)
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
