package safechain

import "fmt"

// =============================================================================
// Ensure & Set: nil-safe deep struct initialization and assignment
// =============================================================================

// Ensure initializes a nil pointer to a new zero value and returns it.
// Chain with E() for a compact one-liner.
//
//	var req Request
//	Ensure(&req.Auth)                    // single level
//	Ensure(&Ensure(&req.Auth).Key)       // two levels chained
func Ensure[T any](pp **T) *T {
	if *pp == nil {
		*pp = new(T)
	}
	return *pp
}

// E is a shorthand alias for Ensure. Designed for compact chaining.
//
//	var req Request
//	E(&E(&E(&req.Auth).Key).Session).Token = ptr("hello")
func E[T any](pp **T) *T {
	return Ensure(pp)
}

// Set safely assigns a value deep in a nested struct.
// Returns true on success, false if the path panics.
//
//	var req Request
//	Set(func() *string {
//	    return &E(&E(&E(&req.Auth).Key).Session).Token
//	}, ptr("my_token"))
func Set[T any](path func() *T, val T) bool {
	_, err := SetErr(path, val)
	return err == nil
}

// SetErr is like Set but returns an error describing the failure.
//
//	val, err := SetErr(func() *string {
//	    return &E(&E(&E(&req.Auth).Key).Session).Token
//	}, ptr("my_token"))
func SetErr[T any](path func() *T, val T) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			result = zero
			err = fmt.Errorf("set failed: %v", r)
		}
	}()
	p := path()
	if p == nil {
		return val, fmt.Errorf("set failed: path returned nil pointer")
	}
	*p = val
	return val, nil
}
