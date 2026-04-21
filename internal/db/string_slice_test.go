package db

import "testing"

func TestStringSliceValueAndScan(t *testing.T) {
	original := StringSlice{"backend", "bug", "urgent"}

	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	var decoded StringSlice
	if err := decoded.Scan(value); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(decoded) != len(original) {
		t.Fatalf("decoded length = %d, want %d", len(decoded), len(original))
	}

	for i := range original {
		if decoded[i] != original[i] {
			t.Fatalf("decoded[%d] = %q, want %q", i, decoded[i], original[i])
		}
	}
}

func TestStringSliceScanNilBecomesEmptySlice(t *testing.T) {
	var decoded StringSlice
	if err := decoded.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error = %v", err)
	}

	if decoded == nil || len(decoded) != 0 {
		t.Fatalf("expected empty non-nil slice after Scan(nil), got %#v", decoded)
	}
}
