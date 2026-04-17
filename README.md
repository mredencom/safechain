<p align="center">
  <img src="https://img.shields.io/badge/🔗-safechain-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="safechain logo" />
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/mredencom/safechain"><img src="https://pkg.go.dev/badge/github.com/mredencom/safechain.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/mredencom/safechain"><img src="https://goreportcard.com/badge/github.com/mredencom/safechain" alt="Go Report Card"></a>
  <a href="https://github.com/mredencom/safechain/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
</p>

<p align="center">
  <b>Nil-safe access to deeply nested struct pointer chains in Go.</b><br/>
  No more <code>if a != nil && a.B != nil && a.B.C != nil</code> boilerplate.
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#api">API</a> •
  <a href="#benchmark">Benchmark</a> •
  <a href="./README_CN.md">中文文档</a>
</p>

---

## The Problem

Go has no optional chaining. Accessing `req.A.B.C.D` requires checking every pointer:

```go
var token string
if req != nil && req.Auth != nil && req.Auth.Key != nil &&
    req.Auth.Key.Session != nil && req.Auth.Key.Session.Token != nil {
    token = *req.Auth.Key.Session.Token
}
```

**safechain** eliminates this boilerplate with two approaches:

| Approach | Use When | Overhead |
|----------|----------|----------|
| `Safe` / `Must` / `OrVal` | Don't need to know *which* field is nil | ~4 ns, 0 alloc |
| `Dig` + `S()` | Need precise error: *which* field was nil | ~190 ns/100 depth, 0 alloc |

## Installation

```bash
go get github.com/mredencom/safechain
```

## Quick Start

```go
import "github.com/mredencom/safechain"

// One-liner — returns zero value if any pointer is nil
token := safechain.Must(func() string {
    return *req.Auth.Key.Session.Token
})

// With fallback
token := safechain.OrVal(func() string {
    return *req.Auth.Key.Session.Token
}, "N/A")

// Comma-ok style
token, ok := safechain.Safe(func() string {
    return *req.Auth.Key.Session.Token
})
```

## API

### Core — recover-based (most commonly used)

```go
// Returns (value, ok) — use * to dereference pointer fields like *string
val, ok := Safe(func() string { return *req.Auth.Key.Session.Token })

// Returns value or zero
val := Must(func() string { return *req.Auth.Key.Session.Token })

// Returns value or fallback
val := OrVal(func() string { return *req.Auth.Key.Session.Token }, "N/A")

// For non-pointer fields (e.g. int), no * needed
count, ok := Safe(func() int { return req.Auth.Key.Session.RetryCount })
```

### Logic — And / Or / Any / Not / None / Count / AtLeast / NotNil

```go
// ALL conditions must be true
ok := And(
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    HasPrefix(func() string { return *req.Auth.Key.Session.Token }, "Bearer"),
    Gt(func() int { return *req.Auth.RetryCount }, 0),
)

// At least ONE condition is true (Or is an alias for Any)
ok := Or(
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    Check(func() { _ = *req.Fallback }),
)

// Negate a condition
ok := And(
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    Not(HasPrefix(func() string { return *req.Auth.Key.Session.Token }, "Bearer")),
)

// ALL conditions must be false
ok := None(
    Check(func() { _ = *req.BannedToken }),
    Check(func() { _ = *req.ExpiredToken }),
)

// Count how many conditions are true
n := Count(
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    Check(func() { _ = *req.Fallback }),
    Check(func() { _ = *req.Meta.TraceID }),
)

// At least N conditions must be true
ok := AtLeast(2,
    Check(func() { _ = *req.Auth.Key.Session.Token }),
    Check(func() { _ = *req.Fallback }),
    Check(func() { _ = *req.Meta.TraceID }),
)

// NotNil — simplified nil check, no need for _ = or *
ok := NotNil(func() any { return req.Auth.Key.Session })
```

### Coalesce — First / MustFirst

```go
// First non-nil value wins (like SQL COALESCE)
token := MustFirst(
    func() string { return *req.Auth.Key.Session.Token },
    func() string { return *req.Fallback },
    func() string { return "anonymous" },
)
```

### Comparison — Eq / Ne / Gt / Gte / Lt / Lte / Between

Use `*` to dereference pointer fields (e.g. `*string`, `*int`). For value fields (e.g. `int`, `string`), no `*` needed.

```go
// *r.A.Name — Name is *string, needs *
Eq(func() string { return *r.A.Name }, "admin")
Ne(func() string { return *r.A.Name }, "guest")

// *r.A.Score — Score is *int, needs *
Gt(func() int { return *r.A.Score }, 10)
Gte(func() int { return *r.A.Score }, 10)
Lt(func() float64 { return *r.A.Rate }, 3.14)
Lte(func() float64 { return *r.A.Rate }, 3.14)
Between(func() int { return *r.A.Score }, 1, 100)

// r.A.Count — Count is int (value type), no * needed
Gt(func() int { return r.A.Count }, 0)

// Interval variants
BetweenExcl(func() int { return *r.A.Score }, 0, 100)   // (0, 100)  open
BetweenLExcl(func() int { return *r.A.Score }, 0, 100)  // (0, 100]  left-open
BetweenRExcl(func() int { return *r.A.Score }, 0, 100)  // [0, 100)  right-open

// Custom predicate
Match(func() string { return *r.A.Name }, func(v string) bool { return len(v) > 3 })
```

### String matchers

```go
HasPrefix(func() string { return *r.A.Name }, "hello")
HasSuffix(func() string { return *r.A.Name }, "world")
Contains(func() string { return *r.A.Name }, "llo_wor")
EqFold(func() string { return *r.A.Name }, "HELLO")
MatchRegexp(func() string { return *r.A.Name }, `^\d+$`)
MatchRegexpCompiled(func() string { return *r.A.Name }, re)  // pre-compiled, for hot paths
```

### []byte matchers

```go
BytesHasPrefix(func() []byte { return *r.A.Data }, []byte("hello"))
BytesHasSuffix(func() []byte { return *r.A.Data }, []byte("world"))
BytesContains(func() []byte { return *r.A.Data }, []byte("llo"))
BytesEq(func() []byte { return *r.A.Data }, []byte("exact"))
BytesMatchRegexp(func() []byte { return *r.A.Data }, `\d+`)
BytesMatchRegexpCompiled(func() []byte { return *r.A.Data }, re)
```

### SafeDig — named chain with precise error reporting

No `unsafe.Pointer` needed. Use `F()` to define each step:

```go
// MustSafeDig — just check existence, returns bool
ok := MustSafeDig(req,
    F("Auth", func(r *Request) any { return r.Auth }),
    F("Key", func(a *Auth) any { return a.Key }),
    F("Session", func(k *Key) any { return k.Session }),
)

// SafeDig — get value + bool
token, ok := SafeDig[string](req,
    F("Auth", func(r *Request) any { return r.Auth }),
    F("Key", func(a *Auth) any { return a.Key }),
    F("Session", func(k *Key) any { return k.Session }),
    F("Token", func(s *Session) any { return s.Token }),
)

// SafeDigErr — get value + *NilError pinpointing the nil field
token, err := SafeDigErr[string](req,
    F("Auth", func(r *Request) any { return r.Auth }),
    F("Key", func(a *Auth) any { return a.Key }),
)
// err: nil pointer at field "Key" in path "Auth.Key"
```

### Dig — zero-alloc advanced version (unsafe.Pointer)

For hot paths where allocation matters:

```go
import "unsafe"

val, err := safechain.Dig[string](req,
    safechain.S("Auth", func(r *Request) unsafe.Pointer { return unsafe.Pointer(r.Auth) }),
    safechain.S("Key", func(a *Auth) unsafe.Pointer { return unsafe.Pointer(a.Key) }),
    safechain.S("Session", func(k *Key) unsafe.Pointer { return unsafe.Pointer(k.Session) }),
    safechain.S("Token", func(s *Session) unsafe.Pointer { return unsafe.Pointer(s.Token) }),
)
```

### Composing with And / Or

All comparison and matcher functions return `bool`, so they plug directly into `And`/`Or`:

```go
ok := And(
    HasPrefix(func() string { return *req.Auth.Key.Session.Token }, "Bearer"),
    Gt(func() int { return *req.Auth.RetryCount }, 0),
    Check(func() { _ = *req.Meta.TraceID }),
)
```

### Transform — Map / MustMap

```go
// Safely get a value and transform it
upper, ok := Map(func() string { return *r.A.Name }, strings.ToUpper)
upper := MustMap(func() string { return *r.A.Name }, strings.ToUpper)
length := MustMap(func() string { return *r.A.Name }, func(s string) int { return len(s) })
```

### Set membership — In / NotIn

```go
In(func() string { return *r.A.Role }, "admin", "superadmin")
NotIn(func() string { return *r.A.Role }, "banned", "suspended")
```

### Zero value — IsZero / NotZero

```go
IsZero(func() string { return *r.A.Name })    // *string: "" → true, "abc" → false
NotZero(func() int { return r.A.Count })       // int (value type): 0 → false, 5 → true
```

### Length — Len / MustLen

```go
n, ok := Len(func() string { return *r.A.Name })
n := MustLen(func() []byte { return *r.A.Data })
```

### Side effect — IfOk

```go
IfOk(func() string { return *r.A.Token }, func(token string) {
    fmt.Println("got token:", token)
})
```

### Error — SafeErr

```go
// Like Safe but returns error instead of bool
val, err := SafeErr(func() string { return *r.A.Name })
// err: "nil pointer dereference: runtime error: ..."
```

## Benchmark

Tested on Apple M4 (10 cores), Go 1.22+.

### Single-goroutine — per function (4-level struct)

| Function | ns/op | allocs |
|----------|-------|--------|
| `Safe` | 7.0 | 0 |
| `Must` | 8.0 | 0 |
| `OrVal` | 8.3 | 0 |
| `Check` | 8.9 | 0 |
| `NotNil` | 8.2 | 0 |
| `Eq` | 9.1 | 0 |
| `Gt` | 7.8 | 0 |
| `Between` | 8.2 | 0 |
| `In` (3 values) | 13 | 0 |
| `Match` | 9.4 | 0 |
| `HasPrefix` | 12 | 0 |
| `Contains` | 13 | 0 |
| `Map` | 9.5 | 0 |
| `First` | 9.7 | 0 |
| `SafeErr` | 7.0 | 0 |
| `And` (2 checks) | 18 | 0 |
| `SafeDig` (4 levels) | 141 | 5 |
| `MustSafeDig` (3 levels) | 110 | 4 |
| `Dig` (4 levels) | 45 | 1 |

### Parallel — concurrent throughput (10 goroutines)

| Function | ns/op | allocs |
|----------|-------|--------|
| `Safe` | 2.2 | 0 |
| `Must` | 2.2 | 0 |
| `Check` | 2.5 | 0 |
| `Eq` | 2.3 | 0 |
| `HasPrefix` | 2.8 | 0 |
| `In` | 3.2 | 0 |
| `Dig` (4 levels) | 43 | 1 |
| `SafeDig` (4 levels) | 102 | 5 |

### Deep nesting — 100 levels

| Approach | Single | Parallel | Allocs |
|----------|--------|----------|--------|
| Manual `if != nil` | 93 ns | 16 ns | 0 |
| `Safe` (recover) | 88 ns | 18 ns | 0 |
| `Dig` (unsafe.Pointer) | 589 ns | 718 ns | 1 |

- `Safe` matches hand-written nil checks at any depth, zero allocation.
- All recover-based functions scale linearly with goroutines — no contention.
- `Dig`/`SafeDig` have allocation overhead from the `names` slice, but still sub-microsecond.

## Concurrency

All functions are **goroutine-safe** — no shared state, all operations are stack-local. Verified with 1000-goroutine stress tests + `-race` detector on every public function:

`Safe`, `Must`, `OrVal`, `Check`, `NotNil`, `And`, `Or`, `Eq`, `Gt`, `Between`, `In`, `Match`, `HasPrefix`, `Contains`, `Map`, `First`, `SafeErr`, `Dig`, `SafeDig`, `MustSafeDig`

The only requirement is that the struct being accessed is not concurrently modified by another goroutine (same as hand-written `if != nil`).

## Requirements

- Go 1.22+

## License

[MIT](LICENSE)
