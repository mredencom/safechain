package safechain

import (
	"regexp"
	"testing"
)

type StrRoot struct {
	A *StrLevelA
}
type StrLevelA struct {
	Name *string
	Data *[]byte
}

func strRoot(name string, data []byte) *StrRoot {
	return &StrRoot{A: &StrLevelA{Name: &name, Data: &data}}
}

func nilStrRoot() *StrRoot { return &StrRoot{A: nil} }

// =============================================================================
// String matchers
// =============================================================================

func TestHasPrefix(t *testing.T) {
	r := strRoot("hello_world", nil)
	if !HasPrefix(func() string { return *r.A.Name }, "hello") {
		t.Error("expected true")
	}
	if HasPrefix(func() string { return *r.A.Name }, "world") {
		t.Error("expected false")
	}
}

func TestHasPrefix_Nil(t *testing.T) {
	r := nilStrRoot()
	if HasPrefix(func() string { return *r.A.Name }, "hello") {
		t.Error("nil path should return false")
	}
}

func TestHasSuffix(t *testing.T) {
	r := strRoot("hello_world", nil)
	if !HasSuffix(func() string { return *r.A.Name }, "world") {
		t.Error("expected true")
	}
	if HasSuffix(func() string { return *r.A.Name }, "hello") {
		t.Error("expected false")
	}
}

func TestHasSuffix_Nil(t *testing.T) {
	r := nilStrRoot()
	if HasSuffix(func() string { return *r.A.Name }, "world") {
		t.Error("nil path should return false")
	}
}

func TestContains(t *testing.T) {
	r := strRoot("hello_world", nil)
	if !Contains(func() string { return *r.A.Name }, "lo_wo") {
		t.Error("expected true")
	}
	if Contains(func() string { return *r.A.Name }, "xyz") {
		t.Error("expected false")
	}
}

func TestContains_Nil(t *testing.T) {
	r := nilStrRoot()
	if Contains(func() string { return *r.A.Name }, "hello") {
		t.Error("nil path should return false")
	}
}

func TestMatchRegexp(t *testing.T) {
	r := strRoot("abc-123-def", nil)
	if !MatchRegexp(func() string { return *r.A.Name }, `^abc-\d+-def$`) {
		t.Error("expected true")
	}
	if MatchRegexp(func() string { return *r.A.Name }, `^xyz`) {
		t.Error("expected false")
	}
}

func TestMatchRegexp_InvalidPattern(t *testing.T) {
	r := strRoot("abc", nil)
	if MatchRegexp(func() string { return *r.A.Name }, `[invalid`) {
		t.Error("invalid pattern should return false")
	}
}

func TestMatchRegexp_Nil(t *testing.T) {
	r := nilStrRoot()
	if MatchRegexp(func() string { return *r.A.Name }, `.*`) {
		t.Error("nil path should return false")
	}
}

func TestMatchRegexpCompiled(t *testing.T) {
	re := regexp.MustCompile(`\d{3}`)
	r := strRoot("abc-123-def", nil)
	if !MatchRegexpCompiled(func() string { return *r.A.Name }, re) {
		t.Error("expected true")
	}
}

func TestEqFold(t *testing.T) {
	r := strRoot("Hello", nil)
	if !EqFold(func() string { return *r.A.Name }, "hello") {
		t.Error("expected true")
	}
	if !EqFold(func() string { return *r.A.Name }, "HELLO") {
		t.Error("expected true")
	}
	if EqFold(func() string { return *r.A.Name }, "world") {
		t.Error("expected false")
	}
}

func TestEqFold_Nil(t *testing.T) {
	r := nilStrRoot()
	if EqFold(func() string { return *r.A.Name }, "hello") {
		t.Error("nil path should return false")
	}
}

// =============================================================================
// []byte matchers
// =============================================================================

func TestBytesHasPrefix(t *testing.T) {
	r := strRoot("", []byte("hello_world"))
	if !BytesHasPrefix(func() []byte { return *r.A.Data }, []byte("hello")) {
		t.Error("expected true")
	}
	if BytesHasPrefix(func() []byte { return *r.A.Data }, []byte("world")) {
		t.Error("expected false")
	}
}

func TestBytesHasPrefix_Nil(t *testing.T) {
	r := nilStrRoot()
	if BytesHasPrefix(func() []byte { return *r.A.Data }, []byte("hello")) {
		t.Error("nil path should return false")
	}
}

func TestBytesHasSuffix(t *testing.T) {
	r := strRoot("", []byte("hello_world"))
	if !BytesHasSuffix(func() []byte { return *r.A.Data }, []byte("world")) {
		t.Error("expected true")
	}
}

func TestBytesContains(t *testing.T) {
	r := strRoot("", []byte("hello_world"))
	if !BytesContains(func() []byte { return *r.A.Data }, []byte("lo_wo")) {
		t.Error("expected true")
	}
}

func TestBytesEq(t *testing.T) {
	r := strRoot("", []byte("abc"))
	if !BytesEq(func() []byte { return *r.A.Data }, []byte("abc")) {
		t.Error("expected true")
	}
	if BytesEq(func() []byte { return *r.A.Data }, []byte("xyz")) {
		t.Error("expected false")
	}
}

func TestBytesEq_Nil(t *testing.T) {
	r := nilStrRoot()
	if BytesEq(func() []byte { return *r.A.Data }, []byte("abc")) {
		t.Error("nil path should return false")
	}
}

func TestBytesMatchRegexp(t *testing.T) {
	r := strRoot("", []byte("abc-123"))
	if !BytesMatchRegexp(func() []byte { return *r.A.Data }, `\d{3}`) {
		t.Error("expected true")
	}
}

func TestBytesMatchRegexpCompiled(t *testing.T) {
	re := regexp.MustCompile(`\d{3}`)
	r := strRoot("", []byte("abc-123"))
	if !BytesMatchRegexpCompiled(func() []byte { return *r.A.Data }, re) {
		t.Error("expected true")
	}
}
