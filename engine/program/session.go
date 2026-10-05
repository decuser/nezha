package program

import (
	"fmt"
	"sync"
	"sync/atomic"

	"nezha/engine/language/grammar"
	"nezha/engine/semantics/evaluator"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
)

// Session is a canonical Nezha program environment retained across source
// submissions. It owns language state (prelude, user environment, evaluator)
// while the supplied Runtime continues to own host effects.
//
// Evaluation is serialized per Session. Interrupt is the only operation meant
// to be called concurrently with Eval; it controls the current evaluation only
// and does not enter the Nezha language or HAL surface.
type Session struct {
	runtime   Runtime
	grammar   *grammar.Grammar
	evaluator *evaluator.Evaluator
	userEnv   *value.Env

	evalMu      sync.Mutex
	stateMu     sync.Mutex
	running     bool
	interrupted atomic.Bool
}

// NewSession constructs a canonical persistent program environment with the
// embedded prelude loaded and an authority-free user scope.
func NewSession(rt Runtime) (*Session, error) {
	if rt == nil {
		return nil, fmt.Errorf("program session: nil runtime")
	}

	g, err := loadGrammar()
	if err != nil {
		return nil, fmt.Errorf("loading grammar: %w", err)
	}

	userEnv, err := newUserEnvironment(g, rt)
	if err != nil {
		return nil, err
	}

	ev := evaluator.New(rt)
	ev.SetGrammar(g)

	s := &Session{
		runtime:   rt,
		grammar:   g,
		evaluator: ev,
		userEnv:   userEnv,
	}
	ev.SetInterruptCheck(s.interrupted.Load)
	return s, nil
}

// Eval evaluates one ordinary Nezha source submission in the persistent user
// environment and returns its ordinary result. Bindings and other user state
// remain available to subsequent submissions.
func (s *Session) Eval(filename, source string) (value.Value, error) {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()

	s.beginEvaluation()
	defer s.endEvaluation()

	lexer := syntax.NewLexer(s.grammar, filename, source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("lexer: %w", err)
	}

	parser := syntax.NewParser(s.grammar, tokens, source)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parser: %w", err)
	}

	s.userEnv.SetFile(filename)
	s.userEnv.SetSource(source)
	result, interrupted := s.evaluator.EvalInterruptible(ast, s.userEnv)
	if interrupted {
		return nil, ErrInterrupted
	}
	if err := executionResultError(result); err != nil {
		return result, err
	}
	return result, nil
}

// Interrupt requests cancellation of the current evaluation. It returns true
// only when an evaluation was active. The Session and its user environment stay
// alive and may be evaluated again immediately after the interrupted Eval
// returns.
func (s *Session) Interrupt() bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if !s.running {
		return false
	}
	s.interrupted.Store(true)
	return true
}

// Reset replaces user state with a fresh canonical prelude/user environment.
// It waits for any active Eval to finish; use Interrupt first when a running
// evaluation must be stopped before reset.
func (s *Session) Reset() error {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()

	userEnv, err := newUserEnvironment(s.grammar, s.runtime)
	if err != nil {
		return err
	}
	s.userEnv = userEnv
	return nil
}

func (s *Session) beginEvaluation() {
	s.stateMu.Lock()
	s.interrupted.Store(false)
	s.running = true
	s.stateMu.Unlock()
}

func (s *Session) endEvaluation() {
	s.stateMu.Lock()
	s.running = false
	s.interrupted.Store(false)
	s.stateMu.Unlock()
}

func newUserEnvironment(g *grammar.Grammar, rt Runtime) (*value.Env, error) {
	preludeEnv := value.NewEnvWithScope(value.ScopePrelude)
	preludeEnv.SetAuthority(rt.PreludeAuthority())
	if err := loadPrelude(g, rt, preludeEnv); err != nil {
		return nil, fmt.Errorf("loading prelude: %w", err)
	}

	userEnv := value.NewEnclosedEnvWithScope(preludeEnv, value.ScopeUser)
	userEnv.SetAuthority(value.NoAuthority())
	return userEnv, nil
}
