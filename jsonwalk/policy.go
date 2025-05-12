package jsonwalk

import (
	"cmp"
	"slices"
	"strings"
)

type mergetOpts struct {
	FilesDir    string
	ResolvePath []string

	DefaultOverWrite bool
	Overwrite        []string
	Append           []string
}

func buildPolicy(c *mergetOpts) *mergePolicy {
	var overwrite []policyEntry[bool]
	for _, pattern := range c.Overwrite {
		overwrite = addPolicy(overwrite, pattern, true)
	}
	for _, pattern := range c.Append {
		overwrite = addPolicy(overwrite, pattern, false)
	}
	// Absolute patterns before relative patterns.
	slices.SortFunc(overwrite, func(a, b policyEntry[bool]) int {
		return cmp.Or(
			compareBool(a.isRelative, b.isRelative),
			cmp.Compare(a.pattern, b.pattern))
	})

	var resolvePaths []policyEntry[bool]
	for _, pattern := range c.ResolvePath {
		resolvePaths = addPolicy(resolvePaths, pattern, true)
	}
	return &mergePolicy{
		overwrite:        overwrite,
		defaultOverwrite: c.DefaultOverWrite,
		resolvePaths:     resolvePaths,
	}
}

type mergePolicy struct {
	overwrite        []policyEntry[bool]
	defaultOverwrite bool
	resolvePaths     []policyEntry[bool]
}

func (m *mergePolicy) isOverwrite(contextPath string) bool {
	for _, entry := range m.overwrite {
		if entry.match(contextPath) {
			return entry.policy
		}
	}
	return m.defaultOverwrite
}

func (m *mergePolicy) resolvePath(contextPath string) bool {
	for _, entry := range m.resolvePaths {
		if entry.match(contextPath) {
			return true
		}
	}
	return false
}

type policyEntry[T comparable] struct {
	pattern    string
	policy     T
	isRelative bool
}

func (e policyEntry[T]) match(contextPath string) bool {
	if e.isRelative && strings.HasSuffix(contextPath, string(e.pattern)) {
		return true
	}
	if e.pattern == contextPath {
		return true
	}
	return false
}

func addPolicy[T comparable](policies []policyEntry[T], pattern string, policy T) []policyEntry[T] {
	if slices.ContainsFunc(policies, func(p policyEntry[T]) bool {
		return p.pattern == pattern && p.policy != policy
	}) {
		panic("config contains conflicting policies")
	}
	if !strings.HasPrefix(pattern, ".") && !strings.HasPrefix(pattern, "$.") {
		pattern = "$." + pattern
	}
	return append(policies, policyEntry[T]{
		pattern:    pattern,
		policy:     policy,
		isRelative: strings.HasPrefix(pattern, "."),
	})
}
func compareBool(a, b bool) int {
	if a == b {
		return 0
	}
	if a {
		return 1
	}
	return -1
}
