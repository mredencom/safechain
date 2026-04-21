// Package safechain provides nil-safe access to deeply nested struct pointer chains in Go.
//
// Core API (recover-based, zero alloc on happy path):
//   - Safe/Must/OrVal — get values from deep pointer chains
//   - And/Or/Any — combine multiple nil checks and matchers
//   - Eq/Ne/Gt/Lt/Between/In/Match — compare values safely
//   - HasPrefix/Contains/MatchRegexp — string and []byte matchers
//   - Map/IfOk/SafeErr — transform, side-effect, error handling
//
// Write API (see set.go):
//   - E/Ensure — auto-allocate nil pointers along nested chains
//   - Set/SetErr — nil-safe deep assignment with panic recovery
//   - Ptr — generic helper for creating pointers inline
//
// Advanced (see dig.go):
//   - SafeDig/SafeDigErr/MustSafeDig + F() — named chain with precise nil field error reporting
//   - Dig + S() — zero-alloc version using unsafe.Pointer (high performance)
package safechain

import (
	"cmp"
	"fmt"
)

// =============================================================================
// 1. Core: get values safely (highest frequency)
// =============================================================================

// Safe executes fn and returns its result.
// If any nil pointer dereference occurs, returns (zero, false).
//
//	val, ok := Safe(func() string {
//	    return *root.LevelA.LevelB.LevelC.Value
//	})
func Safe[T any](fn func() T) (val T, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			val = zero
			ok = false
		}
	}()
	return fn(), true
}

// Must returns the result of fn, or zero value if any pointer is nil.
//
//	val := Must(func() string {
//	    return *root.LevelA.LevelB.LevelC.Value
//	})
func Must[T any](fn func() T) T {
	v, _ := Safe(fn)
	return v
}

// OrVal returns the result of fn, or fallback if any pointer is nil.
//
//	val := OrVal(func() string {
//	    return *root.LevelA.LevelB.LevelC.Value
//	}, "N/A")
func OrVal[T any](fn func() T, fallback T) T {
	v, ok := Safe(fn)
	if !ok {
		return fallback
	}
	return v
}

// Check returns true if fn executes without nil pointer panic.
//
//	ok := Check(func() { _ = *r.A.B.Token })
func Check(fn func()) bool {
	_, ok := Safe(func() struct{} { fn(); return struct{}{} })
	return ok
}

// NotNil is a shorthand for Check — returns true if accessing the value doesn't panic.
// No need for _ = or * dereference.
//
//	ok := NotNil(func() any { return r.A.B.C })
func NotNil(fn func() any) bool {
	_, ok := Safe(fn)
	return ok
}

// =============================================================================
// 2. Logic: combine conditions (high frequency)
// =============================================================================

// And returns true only if ALL conditions are true.
//
//	ok := And(
//	    Check(func() { _ = *r.A.B.C.Value }),
//	    HasPrefix(func() string { return *r.A.B.Name }, "hello"),
//	    Gt(func() int { return *r.A.B.Count }, 5),
//	)
func And(conds ...bool) bool {
	for _, c := range conds {
		if !c {
			return false
		}
	}
	return true
}

// Or is an alias for Any — returns true if at least one condition is true.
//
//	ok := Or(
//	    Check(func() { _ = *r.A.B.Token }),
//	    Check(func() { _ = *r.Fallback }),
//	)
func Or(conds ...bool) bool {
	return Any(conds...)
}

// Any returns true if at least one condition is true.
//
//	ok := Any(
//	    Check(func() { _ = *r.A.B.C.Value }),
//	    Contains(func() string { return *r.A.B.Name }, "token"),
//	)
func Any(conds ...bool) bool {
	for _, c := range conds {
		if c {
			return true
		}
	}
	return false
}

// Not negates a condition. Useful inside And/Or since Go doesn't allow ! on args.
//
//	ok := And(
//	    Check(func() { _ = *r.A.Token }),
//	    Not(HasPrefix(func() string { return *r.A.Token }, "Bearer")),
//	)
func Not(cond bool) bool {
	return !cond
}

// None returns true only if ALL conditions are false.
//
//	ok := None(
//	    Check(func() { _ = *r.A.BannedToken }),
//	    Check(func() { _ = *r.A.ExpiredToken }),
//	)
func None(conds ...bool) bool {
	for _, c := range conds {
		if c {
			return false
		}
	}
	return true
}

// AtLeast returns true if at least n conditions are true.
//
//	ok := AtLeast(2,
//	    Check(func() { _ = *r.A.Token }),
//	    Check(func() { _ = *r.B.Token }),
//	    Check(func() { _ = *r.C.Token }),
//	)
func AtLeast(n int, conds ...bool) bool {
	return Count(conds...) >= n
}

// Count returns the number of conditions that are true.
//
//	n := Count(
//	    Check(func() { _ = *r.A.Token }),
//	    Check(func() { _ = *r.B.Token }),
//	    Check(func() { _ = *r.C.Token }),
//	)
func Count(conds ...bool) int {
	n := 0
	for _, c := range conds {
		if c {
			n++
		}
	}
	return n
}

// =============================================================================
// 3. Comparison: value checks (high frequency)
// =============================================================================

// Eq returns true if fn's result equals expect. Returns false if any pointer is nil.
//
//	ok := Eq(func() string { return *r.A.Name }, "admin")
func Eq[T comparable](fn func() T, expect T) bool {
	v, ok := Safe(fn)
	return ok && v == expect
}

// Ne returns true if fn's result does NOT equal expect. Returns false if any pointer is nil.
func Ne[T comparable](fn func() T, expect T) bool {
	v, ok := Safe(fn)
	return ok && v != expect
}

// Gt returns true if fn's result > threshold. Returns false if any pointer is nil.
func Gt[T cmp.Ordered](fn func() T, threshold T) bool {
	v, ok := Safe(fn)
	return ok && v > threshold
}

// Gte returns true if fn's result >= threshold. Returns false if any pointer is nil.
func Gte[T cmp.Ordered](fn func() T, threshold T) bool {
	v, ok := Safe(fn)
	return ok && v >= threshold
}

// Lt returns true if fn's result < threshold. Returns false if any pointer is nil.
func Lt[T cmp.Ordered](fn func() T, threshold T) bool {
	v, ok := Safe(fn)
	return ok && v < threshold
}

// Lte returns true if fn's result <= threshold. Returns false if any pointer is nil.
func Lte[T cmp.Ordered](fn func() T, threshold T) bool {
	v, ok := Safe(fn)
	return ok && v <= threshold
}

// Between returns true if lo <= value <= hi (closed interval [lo, hi]).
//
//	ok := Between(func() int { return *r.A.Score }, 1, 100)
func Between[T cmp.Ordered](fn func() T, lo, hi T) bool {
	v, ok := Safe(fn)
	return ok && v >= lo && v <= hi
}

// BetweenExcl returns true if lo < value < hi (open interval (lo, hi)).
//
//	ok := BetweenExcl(func() int { return *r.A.Score }, 0, 100)
func BetweenExcl[T cmp.Ordered](fn func() T, lo, hi T) bool {
	v, ok := Safe(fn)
	return ok && v > lo && v < hi
}

// BetweenLExcl returns true if lo < value <= hi (left-open interval (lo, hi]).
//
//	ok := BetweenLExcl(func() int { return *r.A.Score }, 0, 100)
func BetweenLExcl[T cmp.Ordered](fn func() T, lo, hi T) bool {
	v, ok := Safe(fn)
	return ok && v > lo && v <= hi
}

// BetweenRExcl returns true if lo <= value < hi (right-open interval [lo, hi)).
//
//	ok := BetweenRExcl(func() int { return *r.A.Score }, 0, 100)
func BetweenRExcl[T cmp.Ordered](fn func() T, lo, hi T) bool {
	v, ok := Safe(fn)
	return ok && v >= lo && v < hi
}

// In returns true if fn's result is in the given values. Returns false if any pointer is nil.
//
//	ok := In(func() string { return *r.A.Role }, "admin", "superadmin")
func In[T comparable](fn func() T, values ...T) bool {
	v, ok := Safe(fn)
	if !ok {
		return false
	}
	for _, val := range values {
		if v == val {
			return true
		}
	}
	return false
}

// NotIn returns true if fn's result is NOT in the given values. Returns false if any pointer is nil.
//
//	ok := NotIn(func() string { return *r.A.Role }, "banned", "suspended")
func NotIn[T comparable](fn func() T, values ...T) bool {
	v, ok := Safe(fn)
	if !ok {
		return false
	}
	for _, val := range values {
		if v == val {
			return false
		}
	}
	return true
}

// Match returns true if fn's result passes the predicate. Returns false if any pointer is nil.
//
//	ok := Match(func() string { return *r.A.Name }, func(v string) bool { return len(v) > 5 })
func Match[T any](fn func() T, predicate func(T) bool) bool {
	v, ok := Safe(fn)
	return ok && predicate(v)
}

// IsZero returns true if fn's result equals its zero value (e.g. "", 0, false).
// Returns false if any pointer is nil.
//
//	ok := IsZero(func() string { return *r.A.B.Name })
func IsZero[T comparable](fn func() T) bool {
	v, ok := Safe(fn)
	if !ok {
		return false
	}
	var zero T
	return v == zero
}

// NotZero returns true if fn's result is NOT the zero value.
// Returns false if any pointer is nil.
//
//	ok := NotZero(func() string { return *r.A.B.Name })
func NotZero[T comparable](fn func() T) bool {
	v, ok := Safe(fn)
	if !ok {
		return false
	}
	var zero T
	return v != zero
}

// =============================================================================
// 4. Coalesce: pick first non-nil value (medium frequency)
// =============================================================================

// First returns the value from the first non-nil path (like SQL COALESCE).
//
//	val, ok := First(
//	    func() string { return *root.LevelA.LevelB.LevelC.Value },
//	    func() string { return *root.Fallback },
//	    func() string { return "anonymous" },
//	)
func First[T any](fns ...func() T) (T, bool) {
	for _, fn := range fns {
		if v, ok := Safe(fn); ok {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// MustFirst is like First but returns zero value if all paths fail.
func MustFirst[T any](fns ...func() T) T {
	v, _ := First(fns...)
	return v
}

// =============================================================================
// 5. Transform & side-effect (medium frequency)
// =============================================================================

// Map safely gets a value via fn, then applies transform to it.
// Returns (transformed, true) on success, (zero, false) if any pointer is nil.
//
//	upper, ok := Map(
//	    func() string { return *r.A.B.Name },
//	    strings.ToUpper,
//	)
func Map[T, R any](fn func() T, transform func(T) R) (R, bool) {
	v, ok := Safe(fn)
	if !ok {
		var zero R
		return zero, false
	}
	return transform(v), true
}

// MustMap is like Map but returns zero value on failure.
//
//	upper := MustMap(func() string { return *r.A.B.Name }, strings.ToUpper)
func MustMap[T, R any](fn func() T, transform func(T) R) R {
	v, _ := Map(fn, transform)
	return v
}

// IfOk safely gets a value via fn, and if successful, calls action with it.
// Does nothing if any pointer is nil. Returns whether action was called.
//
//	IfOk(func() string { return *r.A.B.Token }, func(token string) {
//	    fmt.Println("got token:", token)
//	})
func IfOk[T any](fn func() T, action func(T)) bool {
	v, ok := Safe(fn)
	if ok {
		action(v)
	}
	return ok
}

// =============================================================================
// 6. Length (lower frequency)
// =============================================================================

// Len safely gets the length of a string or []byte via fn.
// Returns (length, true) on success, (0, false) if any pointer is nil.
//
//	n, ok := Len(func() string { return *r.A.B.Name })
func Len[T string | []byte](fn func() T) (int, bool) {
	v, ok := Safe(fn)
	if !ok {
		return 0, false
	}
	return len(v), true
}

// MustLen is like Len but returns 0 on failure.
func MustLen[T string | []byte](fn func() T) int {
	n, _ := Len(fn)
	return n
}

// =============================================================================
// 7. Error variant (lower frequency)
// =============================================================================

// SafeErr is like Safe but returns an error instead of bool.
// Returns (value, nil) on success, or (zero, error) with the panic message.
//
//	val, err := SafeErr(func() string { return *r.A.B.Name })
func SafeErr[T any](fn func() T) (val T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			val = zero
			err = fmt.Errorf("nil pointer dereference: %v", r)
		}
	}()
	return fn(), nil
}
