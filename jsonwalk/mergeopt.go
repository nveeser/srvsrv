package jsonwalk

type strategy string

const (
	strategyDefault strategy = "default"
	strategyIgnore  strategy = "ignore"
	strategyReplace          = "replace"
	strategyAppend           = "append"
)

type optionTrie struct {
	root *node
}

type node struct {
	segment  string
	children map[string]*node
	isSet    bool
	strategy strategy
}

func (t *optionTrie) put(path Path, strat strategy) {
	if t.root == nil {
		t.root = &node{children: make(map[string]*node)}
	}
	n := t.root
	for _, seg := range path {
		if _, ok := n.children[seg]; !ok {
			n.children[seg] = &node{
				segment:  seg,
				children: make(map[string]*node),
				strategy: strategyDefault,
			}
		}
		n = n.children[seg]
	}
	n.isSet = true
	n.strategy = strat
}

func (t *optionTrie) strategy(path Path) strategy {
	if t.root == nil {
		return strategyDefault
	}
	if strat, ok := t.searchRecursive(t.root, path, 0); ok {
		return strat
	}
	return strategyDefault
}

func (t *optionTrie) searchRecursive(node *node, path Path, index int) (strategy, bool) {
	if index == len(path) {
		return node.strategy, node.isSet
	}
	seg := path[index]
	for _, matchSeg := range []string{seg, "*"} {
		if child, ok := node.children[matchSeg]; ok {
			if s, ok := t.searchRecursive(child, path, index+1); ok {
				return s, true
			}
		}
	}
	return strategyDefault, false
}
