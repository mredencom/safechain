package safechain

import "fmt"

// =============================================================================
// Ensure & Set: nil-safe deep struct initialization and assignment
// =============================================================================

// Ensure initializes a nil pointer to a new zero value.
// Chain multiple calls to build out a deeply nested struct without nil checks.
//
//	var req Request
//	Ensure(&req.Auth)
//	Ensure(&req.Auth.Key)
//	Ensure(&req.Auth.Key.Session)
//	// req.Auth.Key.Session is now fully initialized, no nil checks needed
func Ensure[T any](pp **T) *T {
	if *pp == nil {
		*pp = new(T)
	}
	return *pp
}

// Set safely assigns a value deep in a nested struct.
// Returns true on success, false if the path panics or returns nil.
//
//	var req Request
//	Set(func() *string {
//	    Ensure(&req.Auth)
//	    Ensure(&req.Auth.Key)
//	    return &req.Auth.Key.Token
//	}, "my_token")
func Set[T any](path func() *T, val T) bool {
	_, err := SetErr(path, val)
	return err == nil
}

// SetErr is like Set but returns an error describing the failure.
//
//	val, err := SetErr(func() *string {
//	    Ensure(&req.Auth)
//	    return &req.Auth.Key.Token
//	}, "my_token")
//	// err: nil pointer dereference: runtime error: ...
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
