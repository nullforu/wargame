package handlers

import (
	"encoding/json"
	"testing"
)

func TestOptionalStringUnmarshalJSON(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		var v optionalString
		if err := json.Unmarshal([]byte(`"hello"`), &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !v.Set || v.Value == nil || *v.Value != "hello" {
			t.Fatalf("unexpected value: %+v", v)
		}
	})

	t.Run("null value", func(t *testing.T) {
		var v optionalString
		if err := json.Unmarshal([]byte(`null`), &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !v.Set || v.Value != nil {
			t.Fatalf("unexpected value: %+v", v)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		var v optionalString
		if err := json.Unmarshal([]byte(`123`), &v); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestOptionalInt64UnmarshalJSON(t *testing.T) {
	t.Run("int value", func(t *testing.T) {
		var v optionalInt64
		if err := json.Unmarshal([]byte(`42`), &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !v.Set || v.Value == nil || *v.Value != 42 {
			t.Fatalf("unexpected value: %+v", v)
		}
	})

	t.Run("null value", func(t *testing.T) {
		var v optionalInt64
		if err := json.Unmarshal([]byte(`null`), &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !v.Set || v.Value != nil {
			t.Fatalf("unexpected value: %+v", v)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		var v optionalInt64
		if err := json.Unmarshal([]byte(`"x"`), &v); err == nil {
			t.Fatalf("expected error")
		}
	})
}
