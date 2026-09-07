package value

import "nezha/engine/syntax/position"

// TailCall is an internal value used by the evaluator to implement proper tail calls.
// It is not intended to be observed by user programs.
type TailCall struct {
	Fn   Value
	Args []Value
	Pos  position.Position
}

func (t *TailCall) Type() Type { return Type("tailcall") }

func (t *TailCall) Inspect() string { return "<tailcall>" }
