package engine

import "nezha/engine/observe"

// Profiling/observation types live in the leaf observe package. These aliases
// retain the existing engine-facing API while callers migrate independently.
type SemanticKind = observe.SemanticKind

const (
	SemanticArithmetic = observe.SemanticArithmetic
	SemanticComparison = observe.SemanticComparison
	SemanticCall       = observe.SemanticCall
	SemanticIteration  = observe.SemanticIteration
	SemanticIndex      = observe.SemanticIndex
	SemanticSend       = observe.SemanticSend
	SemanticReceive    = observe.SemanticReceive
	SemanticStoreRead  = observe.SemanticStoreRead
	SemanticStoreWrite = observe.SemanticStoreWrite
)

type SemanticSite = observe.SemanticSite
type SemanticProbe = observe.SemanticProbe
type AttributionProbe = observe.AttributionProbe

// EvalProbe receives execution-entry observations for AST nodes. The evaluator
// first offers the node's cached binding; only an unbound node requires source
// identity resolution. This keeps the steady-state Eval path integer-only.
type EvalProbe interface {
	Hit(binding *observe.NodeBinding) bool
	BindAndHit(node string, site SemanticSite, binding *observe.NodeBinding)
}
type SemanticCounts = observe.SemanticCounts
type NumberRealizationCounts = observe.NumberRealizationCounts
type CallRealizationCounts = observe.CallRealizationCounts
type ListRealizationCounts = observe.ListRealizationCounts
type ListRealizationProbe = observe.ListRealizationProbe
type EnvKind = observe.EnvKind

const (
	EnvKindRoot     = observe.EnvKindRoot
	EnvKindEnclosed = observe.EnvKindEnclosed
	EnvKindCall     = observe.EnvKindCall
	EnvKindIsolated = observe.EnvKindIsolated
)

type EnvRealizationCounts = observe.EnvRealizationCounts
type EnvRealizationProbe = observe.EnvRealizationProbe
type SemanticSiteCount = observe.SemanticSiteCount
type SemanticMeasurement = observe.SemanticMeasurement
type ProfileLabels = observe.ProfileLabels
