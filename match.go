package safechain

import (
	"bytes"
	"regexp"
	"strings"
)

// =============================================================================
// String matchers
// =============================================================================

// HasPrefix returns true if the string at fn starts with prefix. Nil-safe.
func HasPrefix(fn func() string, prefix string) bool {
	v, ok := Safe(fn)
	return ok && strings.HasPrefix(v, prefix)
}

// HasSuffix returns true if the string at fn ends with suffix. Nil-safe.
func HasSuffix(fn func() string, suffix string) bool {
	v, ok := Safe(fn)
	return ok && strings.HasSuffix(v, suffix)
}

// Contains returns true if the string at fn contains substr. Nil-safe.
func Contains(fn func() string, substr string) bool {
	v, ok := Safe(fn)
	return ok && strings.Contains(v, substr)
}

// MatchRegexp returns true if the string at fn matches the regex pattern. Nil-safe.
// Returns false if the path is nil or the pattern is invalid.
func MatchRegexp(fn func() string, pattern string) bool {
	v, ok := Safe(fn)
	if !ok {
		return false
	}
	matched, err := regexp.MatchString(pattern, v)
	return err == nil && matched
}

// MatchRegexpCompiled is like MatchRegexp but takes a pre-compiled *regexp.Regexp.
// Use this in hot paths to avoid re-compiling the pattern on every call.
func MatchRegexpCompiled(fn func() string, re *regexp.Regexp) bool {
	v, ok := Safe(fn)
	return ok && re.MatchString(v)
}

// EqFold returns true if the string at fn equals target under Unicode case-folding. Nil-safe.
func EqFold(fn func() string, target string) bool {
	v, ok := Safe(fn)
	return ok && strings.EqualFold(v, target)
}

// =============================================================================
// []byte matchers
// =============================================================================

// BytesHasPrefix returns true if the []byte at fn starts with prefix. Nil-safe.
func BytesHasPrefix(fn func() []byte, prefix []byte) bool {
	v, ok := Safe(fn)
	return ok && bytes.HasPrefix(v, prefix)
}

// BytesHasSuffix returns true if the []byte at fn ends with suffix. Nil-safe.
func BytesHasSuffix(fn func() []byte, suffix []byte) bool {
	v, ok := Safe(fn)
	return ok && bytes.HasSuffix(v, suffix)
}

// BytesContains returns true if the []byte at fn contains subslice. Nil-safe.
func BytesContains(fn func() []byte, subslice []byte) bool {
	v, ok := Safe(fn)
	return ok && bytes.Contains(v, subslice)
}

// BytesEq returns true if the []byte at fn equals expect. Nil-safe.
func BytesEq(fn func() []byte, expect []byte) bool {
	v, ok := Safe(fn)
	return ok && bytes.Equal(v, expect)
}

// BytesMatchRegexp returns true if the []byte at fn matches the regex pattern. Nil-safe.
func BytesMatchRegexp(fn func() []byte, pattern string) bool {
	v, ok := Safe(fn)
	if !ok {
		return false
	}
	matched, err := regexp.Match(pattern, v)
	return err == nil && matched
}

// BytesMatchRegexpCompiled is like BytesMatchRegexp but takes a pre-compiled *regexp.Regexp.
func BytesMatchRegexpCompiled(fn func() []byte, re *regexp.Regexp) bool {
	v, ok := Safe(fn)
	return ok && re.Match(v)
}
