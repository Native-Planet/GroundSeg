package configparse

import "testing"

func TestBool(t *testing.T) {
	got, err := Bool("enabled", true)
	if err != nil {
		t.Fatalf("Bool returned error for bool input: %v", err)
	}
	if !got {
		t.Fatal("expected parsed bool true")
	}

	if _, err := Bool("enabled", "true"); err == nil {
		t.Fatal("expected Bool to reject non-bool input")
	}
}

func TestInt(t *testing.T) {
	gotInt, err := Int("retries", 3)
	if err != nil {
		t.Fatalf("Int returned error for int input: %v", err)
	}
	if gotInt != 3 {
		t.Fatalf("expected parsed int 3, got %d", gotInt)
	}

	gotFloat, err := Int("retries", 7.9)
	if err != nil {
		t.Fatalf("Int returned error for float64 input: %v", err)
	}
	if gotFloat != 7 {
		t.Fatalf("expected float64 input to truncate to 7, got %d", gotFloat)
	}

	if _, err := Int("retries", "3"); err == nil {
		t.Fatal("expected Int to reject non-numeric input")
	}
}

func TestString(t *testing.T) {
	got, err := String("name", "groundseg")
	if err != nil {
		t.Fatalf("String returned error for string input: %v", err)
	}
	if got != "groundseg" {
		t.Fatalf("expected parsed string %q, got %q", "groundseg", got)
	}

	if _, err := String("name", 123); err == nil {
		t.Fatal("expected String to reject non-string input")
	}
}
