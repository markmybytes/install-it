package errcode

import (
	"encoding/json"
	"testing"
)

func TestNew(t *testing.T) {
	e := New("errSaveFailed")
	if e == nil {
		t.Fatal("New returned nil")
	}
	if e.Error() != "errSaveFailed" {
		t.Errorf("Error() = %q, want %q", e.Error(), "errSaveFailed")
	}
	if e.params != nil {
		t.Errorf("params = %v, want nil", e.params)
	}
}

func TestNewf(t *testing.T) {
	e := Newf("errCancelFailed", map[string]any{"name": "foo"})
	if e == nil {
		t.Fatal("Newf returned nil")
	}
	if e.Error() != "errCancelFailed" {
		t.Errorf("Error() = %q, want %q", e.Error(), "errCancelFailed")
	}
	if e.params["name"] != "foo" {
		t.Errorf("params[%q] = %v, want %q", "name", e.params["name"], "foo")
	}
}

// Newf must copy the input map — mutating the caller's map after construction
// must not change the error (immutability contract).
func TestNewfCopiesParams(t *testing.T) {
	params := map[string]any{"name": "foo"}
	e := Newf("errCancelFailed", params)
	params["name"] = "bar"
	if got := e.params["name"]; got != "foo" {
		t.Errorf("params[%q] = %v, want %q (input map mutated the error)", "name", got, "foo")
	}
}

func TestErrorReturnsCode(t *testing.T) {
	e := New("warnNoHardwareInfo")
	if got := e.Error(); got != "warnNoHardwareInfo" {
		t.Errorf("Error() = %q, want %q", got, "warnNoHardwareInfo")
	}
}

func TestMarshalJSONWithParams(t *testing.T) {
	e := Newf("errCancelFailed", map[string]any{"name": "foo"})
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}
	want := `{"code":"errCancelFailed","params":{"name":"foo"}}`
	if string(b) != want {
		t.Errorf("MarshalJSON = %s, want %s", b, want)
	}
}

func TestMarshalJSONWithoutParams(t *testing.T) {
	e := New("errSaveFailed")
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}
	want := `{"code":"errSaveFailed"}`
	if string(b) != want {
		t.Errorf("MarshalJSON = %s, want %s", b, want)
	}
}

// Empty params map must serialize without a params field (omitempty).
func TestMarshalJSONEmptyParams(t *testing.T) {
	e := Newf("errSaveFailed", map[string]any{})
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}
	want := `{"code":"errSaveFailed"}`
	if string(b) != want {
		t.Errorf("MarshalJSON = %s, want %s", b, want)
	}
}

// MarshalJSON must never error — the Wails dispatcher calls log.Fatal on
// marshal failure. Guard against regressions.
func TestMarshalJSONNeverErrors(t *testing.T) {
	for _, e := range []*Error{
		New("errSaveFailed"),
		Newf("errCancelFailed", map[string]any{"name": "foo"}),
		Newf("errCancelFailed", map[string]any{}),
	} {
		if _, err := json.Marshal(e); err != nil {
			t.Errorf("MarshalJSON errored for %q: %v", e.Error(), err)
		}
	}
}