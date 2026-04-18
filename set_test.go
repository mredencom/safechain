package safechain

import "testing"

func TestEnsure_Nil(t *testing.T) {
	var root Root
	Ensure(&root.LevelA)
	if root.LevelA == nil {
		t.Error("expected non-nil")
	}
}

func TestEnsure_NonNil(t *testing.T) {
	original := &LevelA{}
	root := &Root{LevelA: original}
	Ensure(&root.LevelA)
	if root.LevelA != original {
		t.Error("should not replace existing pointer")
	}
}

func TestEnsure_Chain(t *testing.T) {
	var root Root
	Ensure(&root.LevelA)
	Ensure(&root.LevelA.LevelB)
	Ensure(&root.LevelA.LevelB.LevelC)
	if root.LevelA.LevelB.LevelC == nil {
		t.Error("full chain should be initialized")
	}
}

func TestEnsure_ReturnValue(t *testing.T) {
	var root Root
	a := Ensure(&root.LevelA)
	if a != root.LevelA {
		t.Error("should return the pointer")
	}
}

func TestSet_CreateAndAssign(t *testing.T) {
	var root Root
	ok := Set(func() *string {
		Ensure(&root.LevelA)
		Ensure(&root.LevelA.LevelB)
		Ensure(&root.LevelA.LevelB.LevelC)
		Ensure(&root.LevelA.LevelB.LevelC.Value)
		return root.LevelA.LevelB.LevelC.Value
	}, "hello")
	if !ok {
		t.Fatal("expected true")
	}
	if *root.LevelA.LevelB.LevelC.Value != "hello" {
		t.Errorf("got %q", *root.LevelA.LevelB.LevelC.Value)
	}
}

func TestSet_Overwrite(t *testing.T) {
	root := fullRoot()
	ok := Set(func() *string {
		return root.LevelA.LevelB.LevelC.Value
	}, "new")
	if !ok || *root.LevelA.LevelB.LevelC.Value != "new" {
		t.Error("overwrite failed")
	}
}

func TestSet_FieldPointer(t *testing.T) {
	var root Root
	ok := Set(func() **string {
		Ensure(&root.LevelA)
		Ensure(&root.LevelA.LevelB)
		Ensure(&root.LevelA.LevelB.LevelC)
		return &root.LevelA.LevelB.LevelC.Value
	}, ptr("token"))
	if !ok || *root.LevelA.LevelB.LevelC.Value != "token" {
		t.Error("set pointer field failed")
	}
}

func TestSet_NilPath(t *testing.T) {
	var root *Root
	ok := Set(func() *string {
		return root.LevelA.LevelB.LevelC.Value // panics
	}, "val")
	if ok {
		t.Error("expected false for nil root")
	}
}

func TestSetErr_Success(t *testing.T) {
	root := fullRoot()
	val, err := SetErr(func() *string {
		return root.LevelA.LevelB.LevelC.Value
	}, "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "ok" {
		t.Errorf("got %q", val)
	}
}

func TestSetErr_Panic(t *testing.T) {
	var root *Root
	_, err := SetErr(func() *string {
		return root.LevelA.LevelB.LevelC.Value
	}, "val")
	if err == nil {
		t.Error("expected error for nil root panic")
	}
}

func TestSetErr_NilPointer(t *testing.T) {
	_, err := SetErr(func() *string {
		return nil
	}, "val")
	if err == nil {
		t.Error("expected error for nil pointer")
	}
}

func TestSet_ThenRead(t *testing.T) {
	var root Root
	Set(func() *string {
		Ensure(&root.LevelA)
		Ensure(&root.LevelA.LevelB)
		Ensure(&root.LevelA.LevelB.LevelC)
		Ensure(&root.LevelA.LevelB.LevelC.Value)
		return root.LevelA.LevelB.LevelC.Value
	}, "round_trip")

	val, ok := Safe(func() string { return *root.LevelA.LevelB.LevelC.Value })
	if !ok || val != "round_trip" {
		t.Errorf("got (%q, %v)", val, ok)
	}
}
