package runner

import (
	"testing"

	"aiki/engine/semantics/evaluator"
)

func TestSemanticProfileSendRecvAcrossSpawn(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	s.Evaluator.Profile = evaluator.NewProfileProbe(false)
	v := s.Eval(`
let ch = channel()
let worker = (out) {
	send(out, 42)
}
spawn(worker, ch)
recv(ch)
`)
	if v == nil || v.Inspect() != "42" {
		t.Fatalf("result: got %v", v)
	}
	if s.Evaluator.Profile.Measurement().Counts.Send != 1 || s.Evaluator.Profile.Measurement().Counts.Receive != 1 {
		t.Fatalf("send/recv: expected 1/1, got %d/%d", s.Evaluator.Profile.Measurement().Counts.Send, s.Evaluator.Profile.Measurement().Counts.Receive)
	}
}

func TestSemanticProfileSelectReceive(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	s.Evaluator.Profile = evaluator.NewProfileProbe(false)
	v := s.Eval(`
let ch = channel()
let worker = (out) {
	send(out, 7)
}
spawn(worker, ch)
select {
	let x = recv(ch) {
		x
	}
}
`)
	if v == nil || v.Inspect() != "7" {
		t.Fatalf("result: got %v", v)
	}
	if s.Evaluator.Profile.Measurement().Counts.Receive != 1 {
		t.Fatalf("select receive: expected 1, got %d", s.Evaluator.Profile.Measurement().Counts.Receive)
	}
}

func TestSemanticProfileSelectDefaultDoesNotReceive(t *testing.T) {
	s, err := NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	s.Evaluator.Profile = evaluator.NewProfileProbe(false)
	v := s.Eval(`
let ch = channel()
select {
	let x = recv(ch) {
		x
	}
	default {
		:idle
	}
}
`)
	if v == nil || v.Inspect() != ":idle" {
		t.Fatalf("result: got %v", v)
	}
	if s.Evaluator.Profile.Measurement().Counts.Receive != 0 {
		t.Fatalf("select default: expected 0 receives, got %d", s.Evaluator.Profile.Measurement().Counts.Receive)
	}
}
