package safechain

import (
	"testing"
	"unsafe"
)

// ============================================================
// Single-goroutine benchmarks
// ============================================================

func BenchmarkSafe(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		Safe(func() string { return *root.LevelA.LevelB.LevelC.Value })
	}
}

func BenchmarkMust(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		Must(func() string { return *root.LevelA.LevelB.LevelC.Value })
	}
}

func BenchmarkOrVal(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		OrVal(func() string { return *root.LevelA.LevelB.LevelC.Value }, "fb")
	}
}

func BenchmarkCheck(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		Check(func() { _ = *root.LevelA.LevelB.LevelC.Value })
	}
}

func BenchmarkNotNil(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		NotNil(func() any { return root.LevelA.LevelB.LevelC })
	}
}

func BenchmarkAnd(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		And(
			Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
			Check(func() { _ = *root.Meta.TraceID }),
		)
	}
}

func BenchmarkOr(b *testing.B) {
	root := &Root{LevelA: nil, Fallback: ptr("fb")}
	for b.Loop() {
		Or(
			Check(func() { _ = *root.LevelA.LevelB.LevelC.Value }),
			Check(func() { _ = *root.Fallback }),
		)
	}
}

func BenchmarkEq(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		Eq(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123")
	}
}

func BenchmarkGt(b *testing.B) {
	r := numRoot(10, 3.14)
	for b.Loop() {
		Gt(func() int { return *r.A.Score }, 5)
	}
}

func BenchmarkBetween(b *testing.B) {
	r := numRoot(10, 3.14)
	for b.Loop() {
		Between(func() int { return *r.A.Score }, 1, 100)
	}
}

func BenchmarkIn(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		In(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123", "xyz", "nope")
	}
}

func BenchmarkMatch(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		Match(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(v string) bool { return len(v) > 3 })
	}
}

func BenchmarkHasPrefix(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		HasPrefix(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc")
	}
}

func BenchmarkContains(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		Contains(func() string { return *root.LevelA.LevelB.LevelC.Value }, "c12")
	}
}

func BenchmarkMap(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		Map(func() string { return *root.LevelA.LevelB.LevelC.Value }, func(s string) int { return len(s) })
	}
}

func BenchmarkFirst(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		First(
			func() string { return *root.LevelA.LevelB.LevelC.Value },
			func() string { return "fallback" },
		)
	}
}

func BenchmarkSafeErr(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		SafeErr(func() string { return *root.LevelA.LevelB.LevelC.Value })
	}
}

func BenchmarkSafeDig(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		SafeDig[string](root,
			F("LevelA", func(r *Root) any { return r.LevelA }),
			F("LevelB", func(a *LevelA) any { return a.LevelB }),
			F("LevelC", func(b *LevelB) any { return b.LevelC }),
			F("Value", func(c *LevelC) any { return c.Value }),
		)
	}
}

func BenchmarkMustSafeDig(b *testing.B) {
	root := fullRoot()
	for b.Loop() {
		MustSafeDig(root,
			F("LevelA", func(r *Root) any { return r.LevelA }),
			F("LevelB", func(a *LevelA) any { return a.LevelB }),
			F("LevelC", func(b *LevelB) any { return b.LevelC }),
		)
	}
}

func BenchmarkDig_4Level(b *testing.B) {
	root := fullRoot()
	steps := valueSteps()
	for b.Loop() {
		Dig[string](root, steps...)
	}
}

// ============================================================
// Parallel (concurrent) benchmarks
// ============================================================

func BenchmarkSafe_Parallel(b *testing.B) {
	root := fullRoot()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Safe(func() string { return *root.LevelA.LevelB.LevelC.Value })
		}
	})
}

func BenchmarkMust_Parallel(b *testing.B) {
	root := fullRoot()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Must(func() string { return *root.LevelA.LevelB.LevelC.Value })
		}
	})
}

func BenchmarkCheck_Parallel(b *testing.B) {
	root := fullRoot()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Check(func() { _ = *root.LevelA.LevelB.LevelC.Value })
		}
	})
}

func BenchmarkEq_Parallel(b *testing.B) {
	root := fullRoot()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Eq(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123")
		}
	})
}

func BenchmarkHasPrefix_Parallel(b *testing.B) {
	root := fullRoot()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			HasPrefix(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc")
		}
	})
}

func BenchmarkIn_Parallel(b *testing.B) {
	root := fullRoot()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			In(func() string { return *root.LevelA.LevelB.LevelC.Value }, "abc123", "xyz")
		}
	})
}

func BenchmarkDig_Parallel(b *testing.B) {
	root := fullRoot()
	steps := valueSteps()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Dig[string](root, steps...)
		}
	})
}

func BenchmarkSafeDig_Parallel(b *testing.B) {
	root := fullRoot()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			SafeDig[string](root,
				F("LevelA", func(r *Root) any { return r.LevelA }),
				F("LevelB", func(a *LevelA) any { return a.LevelB }),
				F("LevelC", func(b *LevelB) any { return b.LevelC }),
				F("Value", func(c *LevelC) any { return c.Value }),
			)
		}
	})
}

// ============================================================
// Deep nesting parallel benchmarks
// ============================================================

func BenchmarkSafe_Depth100_Parallel(b *testing.B) {
	root := buildDeep(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Safe(func() string {
				cur := root
				for i := 0; i < 99; i++ {
					cur = cur.Child
				}
				return *cur.Value
			})
		}
	})
}

func BenchmarkDig_Depth100_Parallel(b *testing.B) {
	root := buildDeep(100)
	steps := buildDeepSteps(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Dig[string](root, steps...)
		}
	})
}

func BenchmarkManual_Depth100_Parallel(b *testing.B) {
	root := buildDeep(100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cur := root
			ok := true
			for j := 0; j < 99; j++ {
				if cur.Child == nil {
					ok = false
					break
				}
				cur = cur.Child
			}
			if ok && cur.Value != nil {
				_ = *cur.Value
			}
		}
	})
}

// Ensure valueSteps and buildDeep/buildDeepSteps use unsafe import
var _ = unsafe.Pointer(nil)
