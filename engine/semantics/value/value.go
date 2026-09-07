// Package value defines the core value types for Nezha.
package value

import (
	"fmt"
	"strings"
	"sync"
)

// Type identifies value kinds.
type Type string

const (
	NumberType   Type = "number"
	BooleanType  Type = "boolean"
	RuneType     Type = "rune"
	StringType   Type = "string"
	SymbolType   Type = "symbol"
	ListType     Type = "list"
	FunctionType Type = "function"
	FaultType    Type = "fault"
	ReturnType   Type = "return"
)

// Value is the interface all Nezha values implement.
type Value interface {
	Type() Type
	Inspect() string
}

// Boolean
type Boolean struct {
	Val bool
}

func (b *Boolean) Type() Type { return BooleanType }
func (b *Boolean) Inspect() string {
	if b.Val {
		return "true"
	}
	return "false"
}

var (
	TRUE  = &Boolean{Val: true}
	FALSE = &Boolean{Val: false}
)

// Rune
type Rune struct {
	Val rune
}

func (r *Rune) Type() Type      { return RuneType }
func (r *Rune) Inspect() string { return fmt.Sprintf("'%c'", r.Val) }

// String
type String struct {
	Val string
}

func (s *String) Type() Type      { return StringType }
func (s *String) Inspect() string { return fmt.Sprintf("%q", s.Val) }

// Symbol
type Symbol struct {
	Val string
}

func (s *Symbol) Type() Type      { return SymbolType }
func (s *Symbol) Inspect() string { return ":" + s.Val }

// List
//
// Elements is the immutable logical-prefix view retained for compatibility
// with existing runtime readers. Adaptive append ownership is private: only
// List methods may extend frontier backing. Callers must not reslice Elements
// beyond len(Elements) or mutate it in place.
type List struct {
	Elements []Value
	Shape    string // empty for raw list, shape name for shaped list

	backing *listBacking
}

type listBacking struct {
	mu       sync.Mutex
	elements []Value // len == allocated slot count
	used     int     // published frontier
}

// ListAppendRealization describes one physical append realization. It is
// implementation feedback for tests, not a Nezha semantic kind.
type ListAppendRealization struct {
	Promoted       bool
	Extended       bool
	Grown          bool
	Forked         bool
	ElementsCopied int
	SlotsAllocated int
}

func (l *List) Type() Type { return ListType }

// Len returns the logical list length.
func (l *List) Len() int { return len(l.Elements) }

// At returns the logical element at i. Bounds behavior intentionally matches
// ordinary Go indexing because evaluator/substrate callers already guard it.
func (l *List) At(i int) Value { return l.Elements[i] }

// LogicalElements returns the immutable logical-prefix view. The returned
// slice is read-only by runtime invariant; it does not grant mutation authority.
func (l *List) LogicalElements() []Value { return l.Elements }

// Append returns a persistent successor list. Flat lists promote by copying
// their logical prefix into runtime-owned frontier storage. A frontier list may
// extend in place only when its logical length equals the published frontier;
// appending from a historical prefix forks to a new backing.
func (l *List) Append(v Value) (*List, ListAppendRealization) {
	if l == nil {
		return &List{Elements: []Value{v}}, ListAppendRealization{Promoted: true, SlotsAllocated: 1}
	}
	if l.backing == nil {
		return promoteListAppend(l, v, false)
	}

	b := l.backing
	b.mu.Lock()
	defer b.mu.Unlock()

	logical := len(l.Elements)
	if logical != b.used {
		return promoteListAppend(l, v, true)
	}

	if b.used < len(b.elements) {
		b.elements[b.used] = v
		b.used++
		return &List{Elements: b.elements[:b.used], Shape: l.Shape, backing: b}, ListAppendRealization{Extended: true}
	}

	newCap := listGrowthCapacity(b.used + 1)
	if doubled := len(b.elements) * 2; doubled > newCap {
		newCap = doubled
	}
	if newCap < 1 {
		newCap = 1
	}
	newElems := make([]Value, newCap)
	copy(newElems, b.elements[:b.used])
	newElems[b.used] = v
	copied := b.used
	b.elements = newElems
	b.used++
	return &List{Elements: b.elements[:b.used], Shape: l.Shape, backing: b}, ListAppendRealization{
		Extended:       true,
		Grown:          true,
		ElementsCopied: copied,
		SlotsAllocated: newCap,
	}
}

func promoteListAppend(l *List, v Value, fork bool) (*List, ListAppendRealization) {
	logical := len(l.Elements)
	capacity := listGrowthCapacity(logical + 1)
	elements := make([]Value, capacity)
	copy(elements, l.Elements)
	elements[logical] = v
	b := &listBacking{elements: elements, used: logical + 1}
	r := ListAppendRealization{
		Promoted:       !fork,
		Forked:         fork,
		ElementsCopied: logical,
		SlotsAllocated: capacity,
	}
	return &List{Elements: elements[:logical+1], Shape: l.Shape, backing: b}, r
}

func listGrowthCapacity(need int) int {
	capacity := 4
	for capacity < need {
		capacity *= 2
	}
	return capacity
}

func (l *List) Inspect() string {
	var parts []string
	if l.Shape != "" {
		parts = append(parts, "@"+l.Shape)
	}
	for _, e := range l.Elements {
		parts = append(parts, e.Inspect())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// Function
type Function struct {
	Name            string
	Params          []string
	Rest            string      // rest parameter name, empty if none
	Body            interface{} // *syntax.Node, but we don't import syntax here
	Env             interface{} // *Env, set at runtime
	TailEnvReusable bool        // body cannot create a closure capturing its call env
}

func (f *Function) Type() Type { return FunctionType }
func (f *Function) Inspect() string {
	if f.Name != "" {
		return fmt.Sprintf("<fn %s>", f.Name)
	}
	return "<fn>"
}

// Callable is implemented by values that can be called as functions.
type Callable interface {
	Value
	Call(args []Value) Value
}

// EMPTY is the empty list singleton, used as "no value".
var EMPTY = &List{Elements: []Value{}}

// Return wraps a return value (internal use).
type Return struct {
	Val Value
}

func (r *Return) Type() Type      { return ReturnType }
func (r *Return) Inspect() string { return r.Val.Inspect() }

// ShapeDef defines a shaped list structure.
type ShapeDef struct {
	Name   string
	Fields []string
}

// Helper functions

func IsTruthy(v Value) bool {
	switch val := v.(type) {
	case *Boolean:
		return val.Val
	case *List:
		// Shaped errors are falsy
		if val.Shape == "error" {
			return false
		}
		return len(val.Elements) > 0
	default:
		return true
	}
}

// Fault represents an internal evaluation failure that halts execution.
// Fault is not an ordinary Nezha value - evaluation halts immediately when one occurs.
// For recoverable errors, use shaped lists: [@error, :kind, "message"]
type Fault struct {
	Message string
	File    string
	Line    int
	Source  string // source line for context
	Stack   []StackFrame
}

func (f *Fault) Type() Type { return FaultType }

// Inspect returns Ruby-style error format matching Error.Inspect.
func (f *Fault) Inspect() string {
	var sb strings.Builder

	// Find the innermost visible frame for the header
	funcName := "<main>"
	if len(f.Stack) > 0 {
		funcName = f.Stack[len(f.Stack)-1].Name
	}

	// First line: file:line:in 'func': message
	if f.File != "" && f.Line > 0 {
		sb.WriteString(fmt.Sprintf("%s:%d:in '%s': %s", f.File, f.Line, funcName, f.Message))
	} else if f.Line > 0 {
		sb.WriteString(fmt.Sprintf("line %d:in '%s': %s", f.Line, funcName, f.Message))
	} else {
		sb.WriteString(f.Message)
	}

	// Second line: source code (indented)
	if f.Source != "" {
		sb.WriteString(fmt.Sprintf("\n    %s", strings.TrimSpace(f.Source)))
	}

	// Stack trace: from file:line:in 'func' (skip innermost, shown above)
	for i := len(f.Stack) - 2; i >= 0; i-- {
		frame := f.Stack[i]
		if frame.File != "" {
			sb.WriteString(fmt.Sprintf("\n        from %s:%d:in '%s'", frame.File, frame.Line, frame.Name))
		} else {
			sb.WriteString(fmt.Sprintf("\n        from line %d:in '%s'", frame.Line, frame.Name))
		}
	}

	return sb.String()
}

func NewFault(format string, args ...interface{}) *Fault {
	return &Fault{Message: fmt.Sprintf(format, args...)}
}

func NewFaultAt(file string, line int, source string, stack []StackFrame, format string, args ...interface{}) *Fault {
	return &Fault{
		Message: fmt.Sprintf(format, args...),
		File:    file,
		Line:    line,
		Source:  source,
		Stack:   stack,
	}
}

// IsFault returns true only for internal halting failures.
func IsFault(v Value) bool {
	_, ok := v.(*Fault)
	return ok
}

// IsShapedError returns true for shaped recoverable errors: [@error, :kind, "message"]
func IsShapedError(v Value) bool {
	list, ok := v.(*List)
	if !ok || list.Shape != "error" {
		return false
	}
	// Must have exactly 2 elements: :kind and "message"
	if len(list.Elements) != 2 {
		return false
	}
	_, kindOk := list.Elements[0].(*Symbol)
	_, msgOk := list.Elements[1].(*String)
	return kindOk && msgOk
}

// NewShapedError creates a shaped recoverable error: [@error, :kind, "message"]
func NewShapedError(kind string, format string, args ...interface{}) *List {
	return &List{
		Shape: "error",
		Elements: []Value{
			&Symbol{Val: kind},
			&String{Val: fmt.Sprintf(format, args...)},
		},
	}
}

// DeepEqual returns true if two values are structurally equal.
// Numbers are compared by value, strings/symbols/runes/booleans by content,
// and lists recursively element-by-element including shape.
func DeepEqual(a, b Value) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch av := a.(type) {
	case *Number:
		bv := b.(*Number)
		return av.Equal(bv)
	case *String:
		bv := b.(*String)
		return av.Val == bv.Val
	case *Symbol:
		bv := b.(*Symbol)
		return av.Val == bv.Val
	case *Boolean:
		bv := b.(*Boolean)
		return av.Val == bv.Val
	case *Rune:
		bv := b.(*Rune)
		return av.Val == bv.Val
	case *List:
		bv := b.(*List)
		if av.Shape != bv.Shape {
			return false
		}
		if len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !DeepEqual(av.Elements[i], bv.Elements[i]) {
				return false
			}
		}
		return true
	default:
		// Functions, channels, modules, and other opaque values — identity only.
		return a == b
	}
}
