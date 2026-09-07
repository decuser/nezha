package runner

import "testing"

func TestSessionResetSignal(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	v := s.Eval(`reset()`)
	if v == nil || v.Type() != "reset" {
		if v == nil {
			t.Fatalf("expected reset value, got nil")
		}
		t.Fatalf("expected reset value, got %s", v.Type())
	}

	if err := s.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
}
