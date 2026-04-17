package safechain

import (
	"fmt"
	"strings"
	"unsafe"
)

// =============================================================================
// SafeDig: friendly API — no unsafe.Pointer for callers (most commonly used)
// =============================================================================

// MustSafeDig returns true if the entire chain resolves without nil.
// Use when you only care about existence, not the value.
//
//	ok := MustSafeDig(req,
//	    F("Auth", func(r *Request) any { return r.Auth }),
//	    F("Key", func(a *Auth) any { return a.Key }),
//	    F("Session", func(k *Key) any { return k.Session }),
//	)
func MustSafeDig(root any, fields ...Field) bool {
	_, err := SafeDigErr[any](root, fields...)
	return err == nil
}

// SafeDig walks a chain of named fields — no unsafe.Pointer needed.
// Returns (value, true) on success, (zero, false) if any field is nil.
//
//	token, ok := SafeDig[string](req,
//	    F("Auth", func(r *Request) any { return r.Auth }),
//	    F("Key", func(a *Auth) any { return a.Key }),
//	)
func SafeDig[T any](root any, fields ...Field) (T, bool) {
	v, err := SafeDigErr[T](root, fields...)
	return v, err == nil
}

// SafeDigErr is like SafeDig but returns *NilError pinpointing the nil field.
//
//	token, err := SafeDigErr[string](req,
//	    F("Auth", func(r *Request) any { return r.Auth }),
//	    F("Key", func(a *Auth) any { return a.Key }),
//	)
//	// err: nil pointer at field "Key" in path "Auth.Key"
func SafeDigErr[T any](root any, fields ...Field) (T, error) {
	var zero T

	if ptrOf(root) == nil {
		name := "root"
		if len(fields) > 0 {
			name = fields[0].Name
		}
		return zero, &NilError{Field: "root", Path: name}
	}

	cur := root
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
		next := f.Get(cur)
		if next == nil || ptrOf(next) == nil {
			return zero, nilErr(names, i)
		}
		cur = next
	}

	if v, ok := cur.(T); ok {
		return v, nil
	}
	if v, ok := cur.(*T); ok {
		if v == nil {
			return zero, nilErr(names, len(names)-1)
		}
		return *v, nil
	}
	return zero, fmt.Errorf("type mismatch: got %T, want %T", cur, zero)
}

// =============================================================================
// Dig: zero-alloc, unsafe.Pointer based (high performance, advanced)
// =============================================================================

// Dig walks a chain of named steps using unsafe.Pointer — zero interface boxing.
// Returns (value, nil) on success, or (zero, *NilError) pinpointing the nil field.
//
//	val, err := Dig[string](root,
//	    S("LevelA", func(r *Root) unsafe.Pointer { return unsafe.Pointer(r.LevelA) }),
//	    S("LevelB", func(a *LevelA) unsafe.Pointer { return unsafe.Pointer(a.LevelB) }),
//	    S("LevelC", func(b *LevelB) unsafe.Pointer { return unsafe.Pointer(b.LevelC) }),
//	    S("Value", func(c *LevelC) unsafe.Pointer { return unsafe.Pointer(c.Value) }),
//	)
//	// err: nil pointer at field "LevelB" in path "LevelA.LevelB"
func Dig[R any](root any, steps ...Step) (R, error) {
	var zero R

	cur := ptrOf(root)
	if cur == nil {
		name := "root"
		if len(steps) > 0 {
			name = steps[0].Name
		}
		return zero, &NilError{Field: "root", Path: name}
	}

	names := make([]string, len(steps))
	for i, s := range steps {
		names[i] = s.Name
		next := s.Fn(cur)
		if next == nil {
			return zero, nilErr(names, i)
		}
		cur = next
	}

	return *(*R)(cur), nil
}

// =============================================================================
// Types
// =============================================================================

// Field describes one level in a SafeDig chain.
type Field struct {
	Name string
	Get  func(any) any
}

// F creates a Field with compile-time type safety on the input.
//
//	F("Auth", func(r *Request) any { return r.Auth })
func F[T any](name string, fn func(T) any) Field {
	return Field{
		Name: name,
		Get: func(v any) any {
			t, ok := v.(T)
			if !ok {
				return nil
			}
			return fn(t)
		},
	}
}

// Step is a named field accessor for use with Dig.
type Step struct {
	Name string
	Fn   func(unsafe.Pointer) unsafe.Pointer
}

// S creates a Step with compile-time type safety on the input.
//
//	S("LevelA", func(r *Root) unsafe.Pointer { return unsafe.Pointer(r.LevelA) })
func S[T any](name string, fn func(*T) unsafe.Pointer) Step {
	return Step{
		Name: name,
		Fn: func(p unsafe.Pointer) unsafe.Pointer {
			return fn((*T)(p))
		},
	}
}

// NilError records which field in the chain was nil.
type NilError struct {
	Field string // the specific field that was nil, e.g. "LevelB"
	Path  string // full path up to the nil field, e.g. "LevelA.LevelB"
}

func (e *NilError) Error() string {
	return fmt.Sprintf("nil pointer at field %q in path %q", e.Field, e.Path)
}

// =============================================================================
// Internal
// =============================================================================

// nilErr builds a *NilError with lazy path construction.
func nilErr(names []string, idx int) *NilError {
	parts := make([]string, idx+1)
	copy(parts, names[:idx+1])
	return &NilError{Field: names[idx], Path: strings.Join(parts, ".")}
}

// iface is the runtime layout of an interface{}.
type iface struct {
	typ unsafe.Pointer
	ptr unsafe.Pointer
}

// ptrOf extracts the data pointer from an interface without allocation.
func ptrOf(v any) unsafe.Pointer {
	return (*iface)(unsafe.Pointer(&v)).ptr
}
