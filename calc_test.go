package demo

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(2, 2); got != 4 {
		t.Fatalf("Add(2,2) = %d, want 4", got)
	}
}

func TestAddNegative(t *testing.T) {
	if got := Add(-3, 1); got != -2 {
		t.Fatalf("Add(-3,1) = %d, want -2", got)
	}
}

func TestMax(t *testing.T) {
	if got := Max(3, 7); got != 7 {
		t.Fatalf("Max(3,7) = %d, want 7", got)
	}
}

func TestMaxEqual(t *testing.T) {
	if got := Max(5, 5); got != 5 {
		t.Fatalf("Max(5,5) = %d, want 5", got)
	}
}
