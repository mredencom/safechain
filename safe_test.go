package safechain

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"unsafe"
)

type Root struct {
	LevelA   *LevelA
	Fallback *string
	Meta     *Meta
}
type LevelA struct{ LevelB *LevelB }
type LevelB struct{ LevelC *LevelC }
type LevelC struct {
	Value    *string
	AltValue *string
}
type Meta struct{ TraceID *string }

func ptr[T any](v T) *T { return &v }

func fullRoot() *Root {
	return &Root{
		LevelA: &LevelA{&LevelB{&LevelC{
			Value:    ptr("abc123"),
			AltValue: ptr("alt456"),
		}}},
		Meta: &Meta{TraceID: ptr("trace-789")},
	}
}

// ============================================================
// Dig tests
// ============================================================

func valueSteps() []Step {
	return []Step{
		S("LevelA", func(r *Root) unsafe.Pointer { return unsafe.Pointer(r.LevelA) }),
		S("LevelB", func(a *LevelA) unsafe.Pointer { return unsafe.Pointer(a.LevelB) }),
		S("LevelC", func(b *LevelB) unsafe.Pointer { return unsafe.Pointer(b.LevelC) }),
		S("Value", func(c *LevelC) unsafe.Pointer { return unsafe.Pointer(c.Value) }),
	}
}

func TestDig_FullPath(t *testing.T) {
	val, err := Dig[string](fullRoot(), valueSteps()...)
	if err != nil || val != "abc123" {
		t.Errorf("got (%q, %v)", val, err)
	}
}

func TestDig_NilRoot(t *testing.T) {
	_, err := Dig[string]((*Root)(nil), valueSteps()...)
	var ne *NilError
	if !errors.As(err, &ne) || ne.Field != "root" {
		t.Errorf("got %v", err)
	}
}

func TestDig_NilMiddle_LevelB(t *testing.T) {
	root := &Root{LevelA: &LevelA{LevelB: nil}}
	_, err := Dig[string](root, valueSteps()...)
	var ne *NilError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NilError, got %v", err)
	}
	if ne.Field != "LevelB" {
		t.Errorf("Field = %q, want LevelB", ne.Field)
	}
	if ne.Path != "LevelA.LevelB" {
		t.Errorf("Path = %q, want LevelA.LevelB", ne.Path)
	}
}

func TestDig_NilMiddle_LevelC(t *testing.T) {
	root := &Root{LevelA: &LevelA{&LevelB{LevelC: nil}}}
	_, err := Dig[string](root, valueSteps()...)
	var ne *NilError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NilError, got %v", err)
	}
	if ne.Field != "LevelC" {
		t.Errorf("Field = %q, want LevelC", ne.Field)
	}
}

func TestDig_NilLeaf(t *testing.T) {
	root := &Root{LevelA: &LevelA{&LevelB{&LevelC{Value: nil}}}}
	_, err := Dig[string](root, valueSteps()...)
	if err == nil {
		t.Fatal("expected error")
	}
	var ne *NilError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NilError, got %v", err)
	}
	if ne.Field != "Value" {
		t.Errorf("Field = %q, want Value", ne.Field)
	}
}

// ============================================================
// Safe / And / Any / First
// ============================================================

func TestSafe_FullPath(t *testing.T) {
	root := fullRoot()
	val, ok := Safe(func() string {
		return *root.LevelA.LevelB.LevelC.Value
	})
	if !ok || val != "abc123" {
		t.Errorf("got (%q, %v)", val, ok)
	}
}

func TestAnd_AllPresent(t *testing.T) {
	root := fullRoot()
	ok := And(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
		Check(func() { _ = *root.Meta.TraceID }),
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestAnd_OneFails(t *testing.T) {
	root := &Root{LevelA: nil, Meta: &Meta{TraceID: ptr("t")}}
	ok := And(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
		Check(func() { _ = *root.Meta.TraceID }),
	)
	if ok {
		t.Error("expected false")
	}
}

func TestAnd_WithMatchers(t *testing.T) {
	root := fullRoot()
	ok := And(
		HasPrefix(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc"),
		Check(func() { _ = *root.Meta.TraceID }),
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestAnd_WithMatcherFails(t *testing.T) {
	root := fullRoot()
	ok := And(
		HasPrefix(func() string { return *root.LevelA.LevelB.LevelC.Value }, "xyz"),
		Check(func() { _ = *root.Meta.TraceID }),
	)
	if ok {
		t.Error("expected false: prefix doesn't match")
	}
}

func TestAny_OnePresent(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: ptr("fb")}
	ok := Any(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
		Check(func() { _ = *root.Fallback }),
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestAny_WithMatchers(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: ptr("fallback_token")}
	ok := Any(
		Contains(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc"),
		HasPrefix(func() string { return *root.Fallback }, "fallback"),
	)
	if !ok {
		t.Error("expected true: second matcher should pass")
	}
}

func TestAny_AllMatchersFail(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: ptr("nope")}
	ok := Any(
		Contains(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc"),
		HasPrefix(func() string { return *root.Fallback }, "xyz"),
	)
	if ok {
		t.Error("expected false")
	}
}

// ============================================================
// Not / None / Count / AtLeast
// ============================================================

func TestNot(t *testing.T) {
	if !Not(false) {
		t.Error("Not(false) should be true")
	}
	if Not(true) {
		t.Error("Not(true) should be false")
	}
}

func TestNot_WithMatcher(t *testing.T) {
	root := fullRoot()
	ok := And(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
		Not(HasPrefix(func() string { return *root.LevelA.LevelB.LevelC.Value }, "xyz")),
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestNone_AllFalse(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: nil}
	ok := None(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
		Check(func() { _ = *root.Fallback }),
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestNone_OneTrue(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: ptr("fb")}
	ok := None(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
		Check(func() { _ = *root.Fallback }),
	)
	if ok {
		t.Error("expected false")
	}
}

func TestNone_Empty(t *testing.T) {
	if !None() {
		t.Error("empty None should be true")
	}
}

func TestCount(t *testing.T) {
	root := fullRoot()
	n := Count(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }), // true
		Check(func() { _ = *root.Meta.TraceID }),               // true
		false,                                                  // false
	)
	if n != 2 {
		t.Errorf("got %d, want 2", n)
	}
}

func TestCount_AllFalse(t *testing.T) {
	if Count(false, false, false) != 0 {
		t.Error("expected 0")
	}
}

func TestAtLeast(t *testing.T) {
	root := fullRoot()
	ok := AtLeast(2,
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }), // true
		Check(func() { _ = *root.Meta.TraceID }),               // true
		false,                                                  // false
	)
	if !ok {
		t.Error("expected true: 2 out of 3")
	}
}

func TestAtLeast_NotEnough(t *testing.T) {
	ok := AtLeast(3, true, true, false)
	if ok {
		t.Error("expected false: only 2 out of 3")
	}
}

func TestAtLeast_Zero(t *testing.T) {
	if !AtLeast(0, false, false) {
		t.Error("AtLeast(0) should always be true")
	}
}

func TestFirst_Coalesce(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: ptr("fb")}
	val := MustFirst(
		func() string { return *root.LevelA.LevelB.LevelC.Value },
		func() string { return *root.Fallback },
	)
	if val != "fb" {
		t.Errorf("got %q", val)
	}
}

// ============================================================
// Or (logical) / OrVal (value with fallback)
// ============================================================

func TestOr_Logical(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: ptr("fb")}
	ok := Or(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }), // false
		Check(func() { _ = *root.Fallback }),                   // true
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestOr_AllFalse(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: nil}
	ok := Or(
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
		Check(func() { _ = *root.Fallback }),
	)
	if ok {
		t.Error("expected false")
	}
}

func TestOrVal_Fallback(t *testing.T) {
	root := &Root{LevelA: nil}
	val := OrVal(func() string { return *root.LevelA.LevelB.LevelC.Value }, "default")
	if val != "default" {
		t.Errorf("got %q", val)
	}
}

func TestOrVal_NoFallback(t *testing.T) {
	root := fullRoot()
	val := OrVal(func() string { return *root.LevelA.LevelB.LevelC.Value }, "default")
	if val != "abc123" {
		t.Errorf("got %q", val)
	}
}

// ============================================================
// Eq / Ne / Match
// ============================================================

func TestEq_Match(t *testing.T) {
	root := fullRoot()
	if !Eq(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123") {
		t.Error("expected Eq true")
	}
	if Eq(func() string { return *root.LevelA.LevelB.LevelC.Value }, "wrong") {
		t.Error("expected Eq false")
	}
}

func TestEq_NilPath(t *testing.T) {
	root := &Root{LevelA: nil}
	if Eq(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123") {
		t.Error("nil path should return false")
	}
}

func TestNe_Match(t *testing.T) {
	root := fullRoot()
	if !Ne(func() string { return *root.LevelA.LevelB.LevelC.Value }, "wrong") {
		t.Error("expected Ne true")
	}
	if Ne(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123") {
		t.Error("expected Ne false for equal values")
	}
}

func TestNe_NilPath(t *testing.T) {
	root := &Root{LevelA: nil}
	if Ne(func() string { return *root.LevelA.LevelB.LevelC.Value }, "anything") {
		t.Error("nil path should return false")
	}
}

// ============================================================
// Gt / Gte / Lt / Lte / Between
// ============================================================

type NumRoot struct {
	A *NumA
}
type NumA struct {
	Score *int
	Rate  *float64
}

func numRoot(score int, rate float64) *NumRoot {
	return &NumRoot{A: &NumA{Score: &score, Rate: &rate}}
}

func TestGt(t *testing.T) {
	r := numRoot(10, 3.14)
	if !Gt(func() int { return *r.A.Score }, 5) {
		t.Error("10 > 5")
	}
	if Gt(func() int { return *r.A.Score }, 10) {
		t.Error("10 not > 10")
	}
	if Gt(func() int { return *r.A.Score }, 20) {
		t.Error("10 not > 20")
	}
}

func TestGt_Nil(t *testing.T) {
	r := &NumRoot{A: nil}
	if Gt(func() int { return *r.A.Score }, 0) {
		t.Error("nil should return false")
	}
}

func TestGte(t *testing.T) {
	r := numRoot(10, 3.14)
	if !Gte(func() int { return *r.A.Score }, 10) {
		t.Error("10 >= 10")
	}
	if !Gte(func() int { return *r.A.Score }, 5) {
		t.Error("10 >= 5")
	}
	if Gte(func() int { return *r.A.Score }, 11) {
		t.Error("10 not >= 11")
	}
}

func TestLt(t *testing.T) {
	r := numRoot(10, 3.14)
	if !Lt(func() int { return *r.A.Score }, 20) {
		t.Error("10 < 20")
	}
	if Lt(func() int { return *r.A.Score }, 10) {
		t.Error("10 not < 10")
	}
}

func TestLte(t *testing.T) {
	r := numRoot(10, 3.14)
	if !Lte(func() int { return *r.A.Score }, 10) {
		t.Error("10 <= 10")
	}
	if Lte(func() int { return *r.A.Score }, 9) {
		t.Error("10 not <= 9")
	}
}

func TestBetween(t *testing.T) {
	r := numRoot(10, 3.14)
	if !Between(func() int { return *r.A.Score }, 1, 100) {
		t.Error("10 in [1,100]")
	}
	if !Between(func() int { return *r.A.Score }, 10, 10) {
		t.Error("10 in [10,10]")
	}
	if Between(func() int { return *r.A.Score }, 11, 20) {
		t.Error("10 not in [11,20]")
	}
}

func TestBetween_Float(t *testing.T) {
	r := numRoot(0, 3.14)
	if !Between(func() float64 { return *r.A.Rate }, 3.0, 4.0) {
		t.Error("3.14 in [3.0, 4.0]")
	}
}

func TestBetween_Nil(t *testing.T) {
	r := &NumRoot{A: nil}
	if Between(func() int { return *r.A.Score }, 0, 100) {
		t.Error("nil should return false")
	}
}

// ============================================================
// BetweenExcl / BetweenLExcl / BetweenRExcl
// ============================================================

func TestBetweenExcl(t *testing.T) {
	r := numRoot(10, 3.14)
	if !BetweenExcl(func() int { return *r.A.Score }, 9, 11) {
		t.Error("10 in (9, 11)")
	}
	if BetweenExcl(func() int { return *r.A.Score }, 10, 20) {
		t.Error("10 not in (10, 20) — lo is exclusive")
	}
	if BetweenExcl(func() int { return *r.A.Score }, 0, 10) {
		t.Error("10 not in (0, 10) — hi is exclusive")
	}
}

func TestBetweenExcl_Nil(t *testing.T) {
	r := &NumRoot{A: nil}
	if BetweenExcl(func() int { return *r.A.Score }, 0, 100) {
		t.Error("nil should return false")
	}
}

func TestBetweenLExcl(t *testing.T) {
	r := numRoot(10, 3.14)
	if !BetweenLExcl(func() int { return *r.A.Score }, 9, 10) {
		t.Error("10 in (9, 10]")
	}
	if BetweenLExcl(func() int { return *r.A.Score }, 10, 20) {
		t.Error("10 not in (10, 20] — lo is exclusive")
	}
	if !BetweenLExcl(func() int { return *r.A.Score }, 9, 11) {
		t.Error("10 in (9, 11]")
	}
}

func TestBetweenRExcl(t *testing.T) {
	r := numRoot(10, 3.14)
	if !BetweenRExcl(func() int { return *r.A.Score }, 10, 11) {
		t.Error("10 in [10, 11)")
	}
	if BetweenRExcl(func() int { return *r.A.Score }, 0, 10) {
		t.Error("10 not in [0, 10) — hi is exclusive")
	}
	if !BetweenRExcl(func() int { return *r.A.Score }, 1, 100) {
		t.Error("10 in [1, 100)")
	}
}

// ============================================================
// Gt/Lt in And/Any
// ============================================================

func TestAnd_WithComparisons(t *testing.T) {
	r := numRoot(10, 3.14)
	ok := And(
		Gt(func() int { return *r.A.Score }, 5),
		Lt(func() float64 { return *r.A.Rate }, 4.0),
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestAny_WithComparisons(t *testing.T) {
	r := numRoot(10, 3.14)
	ok := Any(
		Gt(func() int { return *r.A.Score }, 100),      // false
		Lte(func() float64 { return *r.A.Rate }, 3.14), // true
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestMatch_Predicate(t *testing.T) {
	root := fullRoot()
	if !Match(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(v string) bool {
		return len(v) > 3
	}) {
		t.Error("expected Match true")
	}
	if Match(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(v string) bool {
		return len(v) > 100
	}) {
		t.Error("expected Match false")
	}
}

func TestMatch_NilPath(t *testing.T) {
	root := &Root{LevelA: nil}
	if Match(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(v string) bool {
		return true // predicate always true, but path is nil
	}) {
		t.Error("nil path should return false")
	}
}

// ============================================================
// NotNil (simplified Check)
// ============================================================

func TestNotNil_Present(t *testing.T) {
	root := fullRoot()
	if !NotNil(func() any { return root.LevelA.LevelB.LevelC }) {
		t.Error("expected true")
	}
}

func TestNotNil_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	if NotNil(func() any { return root.LevelA.LevelB }) {
		t.Error("expected false")
	}
}

func TestNotNil_InAnd(t *testing.T) {
	root := fullRoot()
	ok := And(
		NotNil(func() any { return root.LevelA.LevelB.LevelC }),
		NotNil(func() any { return root.Meta.TraceID }),
	)
	if !ok {
		t.Error("expected true")
	}
}

// ============================================================
// Map / MustMap
// ============================================================

func TestMap_Success(t *testing.T) {
	root := fullRoot()
	upper, ok := Map(
		func() string { return *root.LevelA.LevelB.LevelC.Value },
		func(s string) string { return s + "_mapped" },
	)
	if !ok || upper != "abc123_mapped" {
		t.Errorf("got (%q, %v)", upper, ok)
	}
}

func TestMap_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	_, ok := Map(
		func() string { return *root.LevelA.LevelB.LevelC.Value },
		func(s string) int { return len(s) },
	)
	if ok {
		t.Error("expected ok=false")
	}
}

func TestMustMap(t *testing.T) {
	root := fullRoot()
	n := MustMap(
		func() string { return *root.LevelA.LevelB.LevelC.Value },
		func(s string) int { return len(s) },
	)
	if n != 6 {
		t.Errorf("got %d, want 6", n)
	}
}

func TestMustMap_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	n := MustMap(
		func() string { return *root.LevelA.LevelB.LevelC.Value },
		func(s string) int { return len(s) },
	)
	if n != 0 {
		t.Errorf("got %d, want 0", n)
	}
}

// ============================================================
// In / NotIn
// ============================================================

func TestIn(t *testing.T) {
	root := fullRoot()
	if !In(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123", "xyz") {
		t.Error("expected true")
	}
	if In(func() string { return *root.LevelA.LevelB.LevelC.Value }, "nope", "nah") {
		t.Error("expected false")
	}
}

func TestIn_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	if In(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123") {
		t.Error("nil should return false")
	}
}

func TestNotIn(t *testing.T) {
	root := fullRoot()
	if !NotIn(func() string { return *root.LevelA.LevelB.LevelC.Value }, "nope", "nah") {
		t.Error("expected true")
	}
	if NotIn(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123", "xyz") {
		t.Error("expected false")
	}
}

func TestNotIn_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	if NotIn(func() string { return *root.LevelA.LevelB.LevelC.Value }, "anything") {
		t.Error("nil should return false")
	}
}

// ============================================================
// IsZero / NotZero
// ============================================================

func TestIsZero_String(t *testing.T) {
	empty := ""
	root := &Root{LevelA: &LevelA{&LevelB{&LevelC{Value: &empty}}}}
	if !IsZero(func() string { return *root.LevelA.LevelB.LevelC.Value }) {
		t.Error("expected true for empty string")
	}
}

func TestIsZero_NonZero(t *testing.T) {
	root := fullRoot()
	if IsZero(func() string { return *root.LevelA.LevelB.LevelC.Value }) {
		t.Error("expected false for non-empty string")
	}
}

func TestIsZero_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	if IsZero(func() string { return *root.LevelA.LevelB.LevelC.Value }) {
		t.Error("nil should return false")
	}
}

func TestNotZero(t *testing.T) {
	root := fullRoot()
	if !NotZero(func() string { return *root.LevelA.LevelB.LevelC.Value }) {
		t.Error("expected true")
	}
}

func TestNotZero_Zero(t *testing.T) {
	empty := ""
	root := &Root{LevelA: &LevelA{&LevelB{&LevelC{Value: &empty}}}}
	if NotZero(func() string { return *root.LevelA.LevelB.LevelC.Value }) {
		t.Error("expected false for empty string")
	}
}

func TestNotZero_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	if NotZero(func() string { return *root.LevelA.LevelB.LevelC.Value }) {
		t.Error("nil should return false")
	}
}

// ============================================================
// Len / MustLen
// ============================================================

func TestLen_String(t *testing.T) {
	root := fullRoot()
	n, ok := Len(func() string { return *root.LevelA.LevelB.LevelC.Value })
	if !ok || n != 6 {
		t.Errorf("got (%d, %v)", n, ok)
	}
}

func TestLen_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	n, ok := Len(func() string { return *root.LevelA.LevelB.LevelC.Value })
	if ok || n != 0 {
		t.Errorf("got (%d, %v)", n, ok)
	}
}

func TestMustLen(t *testing.T) {
	root := fullRoot()
	if MustLen(func() string { return *root.LevelA.LevelB.LevelC.Value }) != 6 {
		t.Error("expected 6")
	}
}

func TestMustLen_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	if MustLen(func() string { return *root.LevelA.LevelB.LevelC.Value }) != 0 {
		t.Error("expected 0")
	}
}

// ============================================================
// IfOk
// ============================================================

func TestIfOk_Called(t *testing.T) {
	root := fullRoot()
	called := false
	ok := IfOk(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(v string) {
		called = true
		if v != "abc123" {
			t.Errorf("got %q", v)
		}
	})
	if !ok || !called {
		t.Error("expected action to be called")
	}
}

func TestIfOk_NotCalled(t *testing.T) {
	root := &Root{LevelA: nil}
	called := false
	ok := IfOk(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(v string) {
		called = true
	})
	if ok || called {
		t.Error("expected action NOT to be called")
	}
}

// ============================================================
// SafeErr
// ============================================================

func TestSafeErr_Success(t *testing.T) {
	root := fullRoot()
	val, err := SafeErr(func() string { return *root.LevelA.LevelB.LevelC.Value })
	if err != nil || val != "abc123" {
		t.Errorf("got (%q, %v)", val, err)
	}
}

func TestSafeErr_Nil(t *testing.T) {
	root := &Root{LevelA: nil}
	val, err := SafeErr(func() string { return *root.LevelA.LevelB.LevelC.Value })
	if err == nil {
		t.Error("expected error")
	}
	if val != "" {
		t.Errorf("got %q, want empty", val)
	}
}

// ============================================================
// SafeDig / SafeDigErr
// ============================================================

func TestSafeDig_FullPath(t *testing.T) {
	root := fullRoot()
	val, ok := SafeDig[string](root,
		F("LevelA", func(r *Root) any { return r.LevelA }),
		F("LevelB", func(a *LevelA) any { return a.LevelB }),
		F("LevelC", func(b *LevelB) any { return b.LevelC }),
		F("Value", func(c *LevelC) any { return c.Value }),
	)
	if !ok || val != "abc123" {
		t.Errorf("got (%q, %v)", val, ok)
	}
}

func TestSafeDig_NilMiddle(t *testing.T) {
	root := &Root{LevelA: &LevelA{LevelB: nil}}
	val, ok := SafeDig[string](root,
		F("LevelA", func(r *Root) any { return r.LevelA }),
		F("LevelB", func(a *LevelA) any { return a.LevelB }),
		F("LevelC", func(b *LevelB) any { return b.LevelC }),
		F("Value", func(c *LevelC) any { return c.Value }),
	)
	if ok || val != "" {
		t.Errorf("expected (zero, false), got (%q, %v)", val, ok)
	}
}

func TestSafeDigErr_NilMiddle(t *testing.T) {
	root := &Root{LevelA: &LevelA{LevelB: nil}}
	_, err := SafeDigErr[string](root,
		F("LevelA", func(r *Root) any { return r.LevelA }),
		F("LevelB", func(a *LevelA) any { return a.LevelB }),
		F("LevelC", func(b *LevelB) any { return b.LevelC }),
		F("Value", func(c *LevelC) any { return c.Value }),
	)
	var ne *NilError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NilError, got %v", err)
	}
	if ne.Field != "LevelB" {
		t.Errorf("Field = %q, want LevelB", ne.Field)
	}
	if ne.Path != "LevelA.LevelB" {
		t.Errorf("Path = %q, want LevelA.LevelB", ne.Path)
	}
}

func TestSafeDig_NilRoot(t *testing.T) {
	_, ok := SafeDig[string]((*Root)(nil),
		F("LevelA", func(r *Root) any { return r.LevelA }),
	)
	if ok {
		t.Error("expected false")
	}
}

func TestSafeDig_NilLeaf(t *testing.T) {
	root := &Root{LevelA: &LevelA{&LevelB{&LevelC{Value: nil}}}}
	_, ok := SafeDig[string](root,
		F("LevelA", func(r *Root) any { return r.LevelA }),
		F("LevelB", func(a *LevelA) any { return a.LevelB }),
		F("LevelC", func(b *LevelB) any { return b.LevelC }),
		F("Value", func(c *LevelC) any { return c.Value }),
	)
	if ok {
		t.Error("expected false")
	}
}

// ============================================================
// MustSafeDig
// ============================================================

func TestMustSafeDig_AllPresent(t *testing.T) {
	root := fullRoot()
	ok := MustSafeDig(root,
		F("LevelA", func(r *Root) any { return r.LevelA }),
		F("LevelB", func(a *LevelA) any { return a.LevelB }),
		F("LevelC", func(b *LevelB) any { return b.LevelC }),
	)
	if !ok {
		t.Error("expected true")
	}
}

func TestMustSafeDig_NilMiddle(t *testing.T) {
	root := &Root{LevelA: &LevelA{LevelB: nil}}
	ok := MustSafeDig(root,
		F("LevelA", func(r *Root) any { return r.LevelA }),
		F("LevelB", func(a *LevelA) any { return a.LevelB }),
		F("LevelC", func(b *LevelB) any { return b.LevelC }),
	)
	if ok {
		t.Error("expected false")
	}
}

func TestMustSafeDig_InAnd(t *testing.T) {
	root := fullRoot()
	ok := And(
		MustSafeDig(root,
			F("LevelA", func(r *Root) any { return r.LevelA }),
			F("LevelB", func(a *LevelA) any { return a.LevelB }),
		),
		MustSafeDig(root,
			F("Meta", func(r *Root) any { return r.Meta }),
			F("TraceID", func(m *Meta) any { return m.TraceID }),
		),
	)
	if !ok {
		t.Error("expected true")
	}
}

// ============================================================
// Concurrent stress tests (1000 goroutines per function)
// ============================================================

const concurrency = 1000

func runConcurrent(t *testing.T, name string, fn func() error) {
	t.Helper()
	var wg sync.WaitGroup
	errs := make(chan error, concurrency)
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("%s: %v", name, err)
	}
}

func TestConcurrent_Safe(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "Safe", func() error {
		val, ok := Safe(func() string { return *root.LevelA.LevelB.LevelC.Value })
		if !ok || val != "abc123" {
			return fmt.Errorf("got (%q, %v)", val, ok)
		}
		return nil
	})
}

func TestConcurrent_Must(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "Must", func() error {
		val := Must(func() string { return *root.LevelA.LevelB.LevelC.Value })
		if val != "abc123" {
			return fmt.Errorf("got %q", val)
		}
		return nil
	})
}

func TestConcurrent_OrVal(t *testing.T) {
	root := &Root{LevelA: nil}
	runConcurrent(t, "OrVal", func() error {
		val := OrVal(func() string { return *root.LevelA.LevelB.LevelC.Value }, "fb")
		if val != "fb" {
			return fmt.Errorf("got %q", val)
		}
		return nil
	})
}

func TestConcurrent_Check(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "Check", func() error {
		if !Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }) {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_NotNil(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "NotNil", func() error {
		if !NotNil(func() any { return root.LevelA.LevelB.LevelC }) {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_And(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "And", func() error {
		ok := And(
			Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
			Check(func() { _ = *root.Meta.TraceID }),
		)
		if !ok {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_Or(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: ptr("fb")}
	runConcurrent(t, "Or", func() error {
		ok := Or(
			Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
			Check(func() { _ = *root.Fallback }),
		)
		if !ok {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_Eq(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "Eq", func() error {
		if !Eq(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123") {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_Gt(t *testing.T) {
	r := numRoot(10, 3.14)
	runConcurrent(t, "Gt", func() error {
		if !Gt(func() int { return *r.A.Score }, 5) {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_Between(t *testing.T) {
	r := numRoot(10, 3.14)
	runConcurrent(t, "Between", func() error {
		if !Between(func() int { return *r.A.Score }, 1, 100) {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_In(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "In", func() error {
		if !In(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123", "xyz") {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_Match(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "Match", func() error {
		if !Match(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(v string) bool { return len(v) > 3 }) {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_HasPrefix(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "HasPrefix", func() error {
		if !HasPrefix(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc") {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_Contains(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "Contains", func() error {
		if !Contains(func() string { return *root.LevelA.LevelB.LevelC.Value }, "c12") {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}

func TestConcurrent_Map(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "Map", func() error {
		v, ok := Map(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(s string) int { return len(s) })
		if !ok || v != 6 {
			return fmt.Errorf("got (%d, %v)", v, ok)
		}
		return nil
	})
}

func TestConcurrent_First(t *testing.T) {
	root := &Root{LevelA: nil, Fallback: ptr("fb")}
	runConcurrent(t, "First", func() error {
		val := MustFirst(
			func() string { return *root.LevelA.LevelB.LevelC.Value },
			func() string { return *root.Fallback },
		)
		if val != "fb" {
			return fmt.Errorf("got %q", val)
		}
		return nil
	})
}

func TestConcurrent_SafeErr(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "SafeErr", func() error {
		val, err := SafeErr(func() string { return *root.LevelA.LevelB.LevelC.Value })
		if err != nil || val != "abc123" {
			return fmt.Errorf("got (%q, %v)", val, err)
		}
		return nil
	})
}

func TestConcurrent_Dig(t *testing.T) {
	root := fullRoot()
	steps := valueSteps()
	runConcurrent(t, "Dig", func() error {
		val, err := Dig[string](root, steps...)
		if err != nil || val != "abc123" {
			return fmt.Errorf("got (%q, %v)", val, err)
		}
		return nil
	})
}

func TestConcurrent_SafeDig(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "SafeDig", func() error {
		val, ok := SafeDig[string](root,
			F("LevelA", func(r *Root) any { return r.LevelA }),
			F("LevelB", func(a *LevelA) any { return a.LevelB }),
			F("LevelC", func(b *LevelB) any { return b.LevelC }),
			F("Value", func(c *LevelC) any { return c.Value }),
		)
		if !ok || val != "abc123" {
			return fmt.Errorf("got (%q, %v)", val, ok)
		}
		return nil
	})
}

func TestConcurrent_MustSafeDig(t *testing.T) {
	root := fullRoot()
	runConcurrent(t, "MustSafeDig", func() error {
		ok := MustSafeDig(root,
			F("LevelA", func(r *Root) any { return r.LevelA }),
			F("LevelB", func(a *LevelA) any { return a.LevelB }),
			F("LevelC", func(b *LevelB) any { return b.LevelC }),
		)
		if !ok {
			return fmt.Errorf("expected true")
		}
		return nil
	})
}
