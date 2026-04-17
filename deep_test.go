package safechain

import (
	"errors"
	"fmt"
	"testing"
	"unsafe"
)

type DeepNode struct {
	Name  string
	Child *DeepNode
	Value *string
}

func buildDeep(depth int) *DeepNode {
	leaf := &DeepNode{Name: fmt.Sprintf("node_%d", depth-1), Value: ptr("leaf_value")}
	cur := leaf
	for i := depth - 2; i >= 0; i-- {
		cur = &DeepNode{Name: fmt.Sprintf("node_%d", i), Child: cur}
	}
	return cur
}

func buildDeepSteps(depth int) []Step {
	steps := make([]Step, 0, depth)
	for i := 0; i < depth-1; i++ {
		name := fmt.Sprintf("Child[%d]", i)
		steps = append(steps, S(name, func(n *DeepNode) unsafe.Pointer {
			return unsafe.Pointer(n.Child)
		}))
	}
	steps = append(steps, S("Value", func(n *DeepNode) unsafe.Pointer {
		return unsafe.Pointer(n.Value)
	}))
	return steps
}

func TestDeep100_Dig_FullPath(t *testing.T) {
	root := buildDeep(100)
	steps := buildDeepSteps(100)
	val, err := Dig[string](root, steps...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "leaf_value" {
		t.Errorf("got %q", val)
	}
}

func TestDeep100_Dig_NilAt50(t *testing.T) {
	root := buildDeep(100)
	cur := root
	for i := 0; i < 49; i++ {
		cur = cur.Child
	}
	cur.Child = nil

	steps := buildDeepSteps(100)
	_, err := Dig[string](root, steps...)
	if err == nil {
		t.Fatal("expected error")
	}
	var ne *NilError
	if !errors.As(err, &ne) {
		t.Fatalf("expected NilError, got %v", err)
	}
	if ne.Field != "Child[49]" {
		t.Errorf("Field = %q, want Child[49]", ne.Field)
	}
	t.Logf("error: %v", err)
}

func TestDeep100_Safe_FullPath(t *testing.T) {
	root := buildDeep(100)
	val, ok := Safe(func() string {
		cur := root
		for i := 0; i < 99; i++ {
			cur = cur.Child
		}
		return *cur.Value
	})
	if !ok || val != "leaf_value" {
		t.Errorf("got (%q, %v)", val, ok)
	}
}

func TestDeep100_Safe_NilAt50(t *testing.T) {
	root := buildDeep(100)
	cur := root
	for i := 0; i < 49; i++ {
		cur = cur.Child
	}
	cur.Child = nil

	_, ok := Safe(func() string {
		c := root
		for i := 0; i < 99; i++ {
			c = c.Child
		}
		return *c.Value
	})
	if ok {
		t.Error("expected ok=false")
	}
}

// ---- Benchmarks ----

func benchmarkDigDepth(b *testing.B, depth int) {
	root := buildDeep(depth)
	steps := buildDeepSteps(depth)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Dig[string](root, steps...)
	}
}

func benchmarkSafeDepth(b *testing.B, depth int) {
	root := buildDeep(depth)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Safe(func() string {
			cur := root
			for i := 0; i < depth-1; i++ {
				cur = cur.Child
			}
			return *cur.Value
		})
	}
}

func benchmarkManualDepth(b *testing.B, depth int) {
	root := buildDeep(depth)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cur := root
		ok := true
		for j := 0; j < depth-1; j++ {
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
}

func BenchmarkDig_Depth10(b *testing.B)     { benchmarkDigDepth(b, 10) }
func BenchmarkDig_Depth50(b *testing.B)     { benchmarkDigDepth(b, 50) }
func BenchmarkDig_Depth100(b *testing.B)    { benchmarkDigDepth(b, 100) }
func BenchmarkSafe_Depth10(b *testing.B)    { benchmarkSafeDepth(b, 10) }
func BenchmarkSafe_Depth50(b *testing.B)    { benchmarkSafeDepth(b, 50) }
func BenchmarkSafe_Depth100(b *testing.B)   { benchmarkSafeDepth(b, 100) }
func BenchmarkManual_Depth10(b *testing.B)  { benchmarkManualDepth(b, 10) }
func BenchmarkManual_Depth50(b *testing.B)  { benchmarkManualDepth(b, 50) }
func BenchmarkManual_Depth100(b *testing.B) { benchmarkManualDepth(b, 100) }
