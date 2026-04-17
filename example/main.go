package main

import (
	"fmt"
	"regexp"

	sc "github.com/mredencom/safechain"
)

// ---- Sample business structs ----

type Request struct {
	Auth     *Auth
	Fallback *string
	Meta     *Meta
}

type Auth struct{ Key *Key }
type Key struct{ Session *Session }
type Session struct {
	Token        *string
	RefreshToken *string
}
type Meta struct {
	TraceID *string
	Tags    *[]byte
}

func ptr[T any](v T) *T { return &v }

func main() {
	fmt.Println("========== safechain examples ==========")
	fmt.Println()

	exampleSafeMustOrVal()
	exampleAndAnyFirst()
	exampleEqMatch()
	exampleStringMatchers()
	exampleBytesMatchers()
	exampleMapInZero()
	exampleIfOkSafeErr()
	exampleDig()
	exampleDigWithSafe()
}

// ---------- 1. Safe / Must / OrVal ----------

func exampleSafeMustOrVal() {
	fmt.Println("--- Safe / Must / OrVal ---")

	req := &Request{Auth: &Auth{&Key{&Session{
		Token: ptr("eyJhbGciOiJIUzI1NiJ9"),
	}}}}

	// Safe: comma-ok style
	token, ok := sc.Safe(func() string { return *req.Auth.Key.Session.Token })
	fmt.Printf("Safe:  token=%q, ok=%v\n", token, ok)

	// Must: get value directly, returns zero on nil
	token = sc.Must(func() string { return *req.Auth.Key.Session.Token })
	fmt.Printf("Must:  token=%q\n", token)

	// OrVal: with fallback
	var nilReq *Request
	token = sc.OrVal(func() string { return *nilReq.Auth.Key.Session.Token }, "anonymous")
	fmt.Printf("OrVal: token=%q (from nil request)\n", token)

	fmt.Println()
}

// ---------- 2. And / Any / First ----------

func exampleAndAnyFirst() {
	fmt.Println("--- And / Any / First ---")

	req := &Request{
		Auth: &Auth{&Key{&Session{
			Token:        ptr("tok_abc"),
			RefreshToken: nil, // intentionally nil
		}}},
		Fallback: ptr("fallback_token"),
		Meta:     &Meta{TraceID: ptr("trace-001")},
	}

	// And: all paths must be non-nil
	ok := sc.And(
		sc.Check(func() { _ = *req.Auth.Key.Session.Token }),
		sc.Check(func() { _ = *req.Meta.TraceID }),
	)
	fmt.Printf("And(token, traceID):        %v\n", ok) // true

	// And: mix nil check + value matching
	ok = sc.And(
		sc.HasPrefix(func() string { return *req.Auth.Key.Session.Token }, "tok_"),
		sc.Check(func() { _ = *req.Meta.TraceID }),
	)
	fmt.Printf("And(prefix+check):          %v\n", ok) // true

	// Any: at least one path is non-nil
	ok = sc.Any(
		sc.Check(func() { _ = *req.Auth.Key.Session.RefreshToken }), // nil
		sc.Check(func() { _ = *req.Fallback }),                      // ok
	)
	fmt.Printf("Any(refreshToken, fallback): %v\n", ok) // true

	// First: like SQL COALESCE
	token := sc.MustFirst(
		func() string { return *req.Auth.Key.Session.RefreshToken }, // nil, skip
		func() string { return *req.Fallback },                      // "fallback_token"
		func() string { return "default" },
	)
	fmt.Printf("First(refresh, fallback):    %q\n", token)

	fmt.Println()
}

// ---------- 3. Eq / Ne / Match ----------

func exampleEqMatch() {
	fmt.Println("--- Eq / Ne / Match ---")

	req := &Request{Auth: &Auth{&Key{&Session{Token: ptr("admin")}}}}

	fmt.Printf("Eq(admin):   %v\n", sc.Eq(func() string { return *req.Auth.Key.Session.Token }, "admin"))
	fmt.Printf("Eq(user):    %v\n", sc.Eq(func() string { return *req.Auth.Key.Session.Token }, "user"))
	fmt.Printf("Ne(user):    %v\n", sc.Ne(func() string { return *req.Auth.Key.Session.Token }, "user"))

	// Match: custom predicate
	fmt.Printf("Match(len>3): %v\n", sc.Match(func() string { return *req.Auth.Key.Session.Token }, func(v string) bool {
		return len(v) > 3
	}))

	// nil path always returns false
	var nilReq *Request
	fmt.Printf("Eq on nil:   %v\n", sc.Eq(func() string { return *nilReq.Auth.Key.Session.Token }, "admin"))

	// Interval variants (using token length as int example)
	getLen := func() int { return len(*req.Auth.Key.Session.Token) }      // "admin" = 5
	fmt.Printf("Between [3,7]:    %v\n", sc.Between(getLen, 3, 7))        // [3,7]  true
	fmt.Printf("BetweenExcl (4,6): %v\n", sc.BetweenExcl(getLen, 4, 6))   // (4,6)  true
	fmt.Printf("BetweenLExcl (5,7]: %v\n", sc.BetweenLExcl(getLen, 5, 7)) // (5,7] false, 5 excluded
	fmt.Printf("BetweenRExcl [5,6): %v\n", sc.BetweenRExcl(getLen, 5, 6)) // [5,6) true

	fmt.Println()
}

// ---------- 4. String matchers ----------

func exampleStringMatchers() {
	fmt.Println("--- String Matchers ---")

	req := &Request{Auth: &Auth{&Key{&Session{Token: ptr("Bearer eyJhbGciOiJIUzI1NiJ9")}}}}
	get := func() string { return *req.Auth.Key.Session.Token }

	fmt.Printf("HasPrefix(Bearer): %v\n", sc.HasPrefix(get, "Bearer"))
	fmt.Printf("HasSuffix(J9):     %v\n", sc.HasSuffix(get, "J9"))
	fmt.Printf("Contains(eyJ):     %v\n", sc.Contains(get, "eyJ"))
	fmt.Printf("EqFold(bearer...): %v\n", sc.EqFold(get, "bearer eyjhbgcioijIUzI1NiJ9"))

	// Regex
	fmt.Printf("MatchRegexp:       %v\n", sc.MatchRegexp(get, `^Bearer\s+.+$`))

	// Pre-compiled regex (recommended for hot paths)
	re := regexp.MustCompile(`eyJ[A-Za-z0-9]+`)
	fmt.Printf("MatchRegexpCompiled: %v\n", sc.MatchRegexpCompiled(get, re))

	fmt.Println()
}

// ---------- 5. []byte matchers ----------

func exampleBytesMatchers() {
	fmt.Println("--- Bytes Matchers ---")

	req := &Request{Meta: &Meta{Tags: ptr([]byte("env:prod;region:us-east-1"))}}
	get := func() []byte { return *req.Meta.Tags }

	fmt.Printf("BytesHasPrefix(env:):    %v\n", sc.BytesHasPrefix(get, []byte("env:")))
	fmt.Printf("BytesHasSuffix(east-1):  %v\n", sc.BytesHasSuffix(get, []byte("east-1")))
	fmt.Printf("BytesContains(region):   %v\n", sc.BytesContains(get, []byte("region")))
	fmt.Printf("BytesEq:                 %v\n", sc.BytesEq(get, []byte("env:prod;region:us-east-1")))
	fmt.Printf("BytesMatchRegexp:        %v\n", sc.BytesMatchRegexp(get, `region:[\w-]+`))

	fmt.Println()
}

// ---------- 6. Map / In / IsZero / Len ----------

func exampleMapInZero() {
	fmt.Println("--- Map / In / IsZero / Len ---")

	req := &Request{Auth: &Auth{&Key{&Session{Token: ptr("admin")}}}}

	// Map: safely get value and transform
	upper, ok := sc.Map(func() string { return *req.Auth.Key.Session.Token }, func(s string) string {
		return "[" + s + "]"
	})
	fmt.Printf("Map:      %q, ok=%v\n", upper, ok)

	// MustMap: transform value, returns zero on nil
	length := sc.MustMap(func() string { return *req.Auth.Key.Session.Token }, func(s string) int {
		return len(s)
	})
	fmt.Printf("MustMap:  len=%d\n", length)

	// In / NotIn
	fmt.Printf("In(admin,root):    %v\n", sc.In(func() string { return *req.Auth.Key.Session.Token }, "admin", "root"))
	fmt.Printf("NotIn(guest,ban):  %v\n", sc.NotIn(func() string { return *req.Auth.Key.Session.Token }, "guest", "banned"))

	// IsZero / NotZero
	fmt.Printf("NotZero(token):    %v\n", sc.NotZero(func() string { return *req.Auth.Key.Session.Token }))

	// Len
	n, _ := sc.Len(func() string { return *req.Auth.Key.Session.Token })
	fmt.Printf("Len(token):        %d\n", n)

	fmt.Println()
}

// ---------- 7. IfOk / SafeErr ----------

func exampleIfOkSafeErr() {
	fmt.Println("--- IfOk / SafeErr ---")

	req := &Request{Auth: &Auth{&Key{&Session{Token: ptr("secret")}}}}

	// IfOk: execute callback when value exists
	sc.IfOk(func() string { return *req.Auth.Key.Session.Token }, func(token string) {
		fmt.Printf("IfOk:     got token=%q\n", token)
	})

	// IfOk: nil path, callback not executed
	var nilReq *Request
	called := sc.IfOk(func() string { return *nilReq.Auth.Key.Session.Token }, func(token string) {
		fmt.Println("this should not print")
	})
	fmt.Printf("IfOk nil: called=%v\n", called)

	// SafeErr: returns error
	val, err := sc.SafeErr(func() string { return *nilReq.Auth.Key.Session.Token })
	fmt.Printf("SafeErr:  val=%q, err=%v\n", val, err != nil)

	fmt.Println()
}

// ---------- 8. SafeDig: precise error reporting ----------

func exampleDig() {
	fmt.Println("--- SafeDig (precise error reporting) ---")

	req := &Request{
		Auth: &Auth{Key: nil}, // Key is nil
	}

	val, ok := sc.SafeDig[string](req,
		sc.F("Auth", func(r *Request) any { return r.Auth }),
		sc.F("Key", func(a *Auth) any { return a.Key }),
		sc.F("Session", func(k *Key) any { return k.Session }),
		sc.F("Token", func(s *Session) any { return s.Token }),
	)
	fmt.Printf("val=%q, ok=%v\n", val, ok)
	// Output: val="", ok=false

	// Happy path
	req2 := &Request{Auth: &Auth{&Key{&Session{Token: ptr("ok_token")}}}}
	val, ok = sc.SafeDig[string](req2,
		sc.F("Auth", func(r *Request) any { return r.Auth }),
		sc.F("Key", func(a *Auth) any { return a.Key }),
		sc.F("Session", func(k *Key) any { return k.Session }),
		sc.F("Token", func(s *Session) any { return s.Token }),
	)
	fmt.Printf("val=%q, ok=%v\n", val, ok)
	// Output: val="ok_token", ok=true

	// SafeDigErr for precise error
	_, err := sc.SafeDigErr[string](req,
		sc.F("Auth", func(r *Request) any { return r.Auth }),
		sc.F("Key", func(a *Auth) any { return a.Key }),
	)
	fmt.Printf("err=%v\n", err)
	// Output: err=nil pointer at field "Key" in path "Auth.Key"
}

// ---------- 9. SafeDig + Safe combined ----------

func exampleDigWithSafe() {
	fmt.Println("--- SafeDig + Safe (combined usage) ---")

	// Scenario: validate a request — need to know exactly which field is missing,
	// AND do value checks on the fields that exist.

	req := &Request{
		Auth: &Auth{&Key{&Session{
			Token:        ptr("Bearer eyJhbGciOiJIUzI1NiJ9"),
			RefreshToken: nil,
		}}},
		Fallback: ptr("fallback_tok"),
		Meta:     &Meta{TraceID: ptr("trace-abc-123"), Tags: ptr([]byte("env:prod"))},
	}

	// Step 1: SafeDig — no unsafe.Pointer needed, precise error on nil
	token, ok := sc.SafeDig[string](req,
		sc.F("Auth", func(r *Request) any { return r.Auth }),
		sc.F("Key", func(a *Auth) any { return a.Key }),
		sc.F("Session", func(k *Key) any { return k.Session }),
		sc.F("Token", func(s *Session) any { return s.Token }),
	)
	fmt.Printf("SafeDig token:  %q, ok=%v\n", token, ok)

	// Step 2: Use Safe matchers to validate the value
	valid := sc.And(
		ok,
		sc.HasPrefix(func() string { return token }, "Bearer"),
		sc.Contains(func() string { return token }, "eyJ"),
	)
	fmt.Printf("Token valid:     %v\n", valid)

	// Step 3: SafeDigErr for precise error when you need it
	_, err := sc.SafeDigErr[string](req,
		sc.F("Auth", func(r *Request) any { return r.Auth }),
		sc.F("Key", func(a *Auth) any { return a.Key }),
		sc.F("Session", func(k *Key) any { return k.Session }),
		sc.F("RefreshToken", func(s *Session) any { return s.RefreshToken }),
	)
	fmt.Printf("SafeDig refresh: err=%v\n", err)

	// Step 4: Fall back using MustFirst when SafeDig fails
	refreshToken := sc.MustFirst(
		func() string { return *req.Auth.Key.Session.RefreshToken }, // nil
		func() string { return *req.Fallback },                      // "fallback_tok"
	)
	fmt.Printf("Fallback:        %q\n", refreshToken)

	// Step 5: SafeDig returns bool, plugs directly into And
	_, traceOk := sc.SafeDig[string](req,
		sc.F("Meta", func(r *Request) any { return r.Meta }),
		sc.F("TraceID", func(m *Meta) any { return m.TraceID }),
	)
	allGood := sc.And(
		traceOk,
		sc.HasPrefix(func() string { return *req.Meta.TraceID }, "trace-"),
		sc.BytesContains(func() []byte { return *req.Meta.Tags }, []byte("prod")),
		sc.Gt(func() int { return len(token) }, 10),
	)
	fmt.Printf("All checks:      %v\n", allGood)

	fmt.Println()
}
