package substrate

import (
	"math/big"

	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

func halFloor(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("floor: want 1 argument, got %d", len(args))
	}
	n, ok := args[0].(*value.Number)
	if !ok {
		return value.NewFault("floor: expected number")
	}
	num := n.Numerator()
	denom := n.Denominator()
	q := new(big.Int)
	rem := new(big.Int)
	q.QuoRem(num, denom, rem)
	if num.Sign() < 0 && rem.Sign() != 0 {
		q.Sub(q, big.NewInt(1))
	}
	return value.NewNumberFromBigInt(q)
}

func halCeil(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("ceil: want 1 argument, got %d", len(args))
	}
	n, ok := args[0].(*value.Number)
	if !ok {
		return value.NewFault("ceil: expected number")
	}
	num := n.Numerator()
	denom := n.Denominator()
	q := new(big.Int)
	rem := new(big.Int)
	q.QuoRem(num, denom, rem)
	if num.Sign() > 0 && rem.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	return value.NewNumberFromBigInt(q)
}

func halTruncate(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("truncate: want 1 argument, got %d", len(args))
	}
	n, ok := args[0].(*value.Number)
	if !ok {
		return value.NewFault("truncate: expected number")
	}
	// Truncate toward zero: just integer division
	num := n.Numerator()
	denom := n.Denominator()
	q := new(big.Int).Quo(num, denom)
	return value.NewNumberFromBigInt(q)
}

func halModulo(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 2 {
		return value.NewFault("modulo: want 2 arguments, got %d", len(args))
	}
	left, ok1 := args[0].(*value.Number)
	right, ok2 := args[1].(*value.Number)
	if !ok1 || !ok2 {
		return value.NewFault("modulo: expected numbers")
	}
	if right.Sign() == 0 {
		// math/native deliberately reaches ordinary Nezha division here so the
		// public fault remains the language-level division-by-zero fault.
		return value.NewFault("division by zero")
	}

	// Match math/native exactly: a - floor(a / abs(b)) * abs(b). Work on
	// semantic exact rationals so compact and promoted Number realizations
	// cross the provider boundary without changing the operation's domain.
	divisor := right.Rat()
	if divisor.Sign() < 0 {
		divisor.Neg(divisor)
	}
	quotient := new(big.Rat).Quo(left.Rat(), divisor)
	q := new(big.Int)
	rem := new(big.Int)
	q.QuoRem(quotient.Num(), quotient.Denom(), rem)
	if quotient.Sign() < 0 && rem.Sign() != 0 {
		q.Sub(q, big.NewInt(1))
	}
	product := new(big.Rat).Mul(new(big.Rat).SetInt(q), divisor)
	result := new(big.Rat).Sub(left.Rat(), product)
	return value.NewNumberFromRat(result)
}
