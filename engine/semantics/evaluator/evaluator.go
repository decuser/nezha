// Package evaluator provides the AST evaluator.
package evaluator

import (
	"fmt"
	"sync"

	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

// handlerFunc is the signature for node handlers.
type handlerFunc func(*Evaluator, *syntax.Node, *value.Env) value.Value

// handlers maps node types to their evaluation functions.
// This is the source of truth for grammar-evaluator coupling.
// Initialized in init() to avoid circular references.
var handlers map[string]handlerFunc

func init() {
	handlers = map[string]handlerFunc{
		// Statements
		"program":     (*Evaluator).evalProgram,
		"statement":   (*Evaluator).evalStatement,
		"let_stmt":    (*Evaluator).evalLet,
		"assign_stmt": (*Evaluator).evalAssign,
		"return_stmt": (*Evaluator).evalReturn,
		"expr_stmt":   (*Evaluator).evalExprStmt,
		"if_stmt":     (*Evaluator).evalIf,
		"while_stmt":  (*Evaluator).evalWhile,
		"match_stmt":  (*Evaluator).evalMatch,
		"block":       (*Evaluator).evalBlock,

		// Expressions
		"expr":         (*Evaluator).evalExpr,
		"pipe_expr":    (*Evaluator).evalExpr,
		"infix_expr":   (*Evaluator).evalExpr,
		"unary_expr":   (*Evaluator).evalExpr,
		"postfix_expr": (*Evaluator).evalExpr,
		"primary":      (*Evaluator).evalPrimary,

		// Literals
		"func_literal": (*Evaluator).evalFunc,
		"list_literal": (*Evaluator).evalList,

		// Tokens
		"NUMBER":   (*Evaluator).evalNumber,
		"STRING":   (*Evaluator).evalString,
		"RUNE":     (*Evaluator).evalRune,
		"SYMBOL":   (*Evaluator).evalSymbol,
		"SHAPE":    (*Evaluator).evalShape,
		"NAME":     (*Evaluator).evalName,
		"TERMINAL": (*Evaluator).evalTerminal,
		"BINOP":    (*Evaluator).evalTerminal,

		// Structural (nil = delegate to single child)
		"call":       nil,
		"index":      nil,
		"access":     nil,
		"params":     nil,
		"param_list": nil,
		"rest_param": nil,
		"pattern":    nil,
		"literal":    nil,
		"field":      nil,
	}
}

// Evaluator evaluates AST nodes.
type Evaluator struct {
	runtime         hal.RuntimeContract
	grammar         *grammar.Grammar
	binaryOperators map[string]struct{}

	// numberLiterals interns semantic realizations of immutable source NUMBER
	// tokens by their literal spelling. sync.Map is required because spawn may
	// evaluate through the same Evaluator concurrently. The key is text rather
	// than *syntax.Node so persistent evaluators do not retain old ASTs.
	numberLiterals sync.Map // map[string]*value.Number

	// argFrames reuses exclusively-owned argument carriers for user functions
	// whose call environments are structurally proven not to escape. sync.Pool
	// keeps acquisition concurrency-safe for spawn without a shared depth index.
	argFrames sync.Pool // *argFrame
}

// New creates an evaluator with a runtime.
func New(runtime hal.RuntimeContract) *Evaluator {
	return &Evaluator{
		runtime: runtime,
	}
}

// SetGrammar sets the grammar and validates handler coverage.
func (e *Evaluator) SetGrammar(g *grammar.Grammar) {
	e.grammar = g
	e.binaryOperators = g.Analysis().TerminalAlternatives("BINOP")
	e.validateHandlers()
	if err := validateBinaryOperatorCoverage(e.binaryOperators, eagerBinaryOperatorSemantics, lazyLogicalOperators); err != nil {
		panic(err)
	}
}

// validateHandlers panics if grammar/evaluator dispatch coverage has drifted.
func (e *Evaluator) validateHandlers() {
	if e.grammar == nil {
		return
	}
	if err := validateHandlerCoverage(e.grammar, handlers); err != nil {
		panic(err)
	}
}

// syntheticHandlerNodes are parser-produced node types that do not correspond
// to a named grammar production or TokenRef. Keep this list explicit. TERMINAL
// nodes are synthesized by engine/syntax/parser.go while matching literal
// terminals; grammar analysis intentionally reports only grammar-produced types.
var syntheticHandlerNodes = map[string]struct{}{
	"TERMINAL": {},
}

func validateHandlerCoverage(g *grammar.Grammar, hs map[string]handlerFunc) error {
	want := g.Analysis().ASTNodeTypes()
	for name := range syntheticHandlerNodes {
		want[name] = struct{}{}
	}

	for name := range want {
		if _, ok := hs[name]; !ok {
			return fmt.Errorf("grammar AST node has no evaluator handler: %s", name)
		}
	}
	for name := range hs {
		if _, ok := want[name]; !ok {
			return fmt.Errorf("evaluator handler has no grammar AST node: %s", name)
		}
	}
	return nil
}

func (e *Evaluator) isBinaryOperator(s string) bool {
	_, ok := e.binaryOperators[s]
	return ok
}

// Eval evaluates an AST node.
func (e *Evaluator) Eval(node *syntax.Node, env *value.Env) value.Value {
	h, ok := handlers[node.Type]
	if !ok {
		// Should never happen after validation
		panic(fmt.Sprintf("no handler for node type: %s", node.Type))
	}

	// nil handler means delegate to single child
	if h == nil {
		if len(node.Children) == 1 {
			return e.Eval(node.Children[0], env)
		}
		return value.EMPTY
	}

	return h(e, node, env)
}
