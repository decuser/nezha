package substrate

import (
	"fmt"
	"sync/atomic"

	"aiki/engine/observe"
	"aiki/engine/runtime/hal"
	"aiki/engine/runtime/hostnative"
	"aiki/engine/semantics/value"
)

// hostNativeOperation is resolved once when an Aiki module binds a host-native
// implementation operation. The returned callable retains this exact function
// pointer; steady-state calls perform no implementation/name lookup or switch.
type hostNativeOperation struct {
	implementation string
	operation      string
	fn             hostnative.Operation
	profileBinding atomic.Uint64
}

func (r *hostNativeOperation) Type() value.Type { return value.FunctionType }
func (r *hostNativeOperation) Inspect() string {
	return fmt.Sprintf("<host-native: %s/%s>", r.implementation, r.operation)
}
func (r *hostNativeOperation) Call(args []value.Value) value.Value { return r.fn(args) }
func (r *hostNativeOperation) NeedsEvalContext() bool              { return false }
func (r *hostNativeOperation) ProfileName() string                 { return r.implementation + "/" + r.operation }
func (r *hostNativeOperation) ProfileNodeSpec() observe.NodeSpec {
	name := r.implementation + "/" + r.operation
	return observe.NodeSpec{
		Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Authority: "runtime/hostnative", Name: name},
		Module:   r.implementation, Layer: "runtime", Boundary: "hostnative", Function: r.operation,
	}
}
func (r *hostNativeOperation) ProfileNodeBinding(registryToken uint32) (uint32, bool) {
	if r == nil || registryToken == 0 {
		return 0, false
	}
	packed := r.profileBinding.Load()
	if uint32(packed>>32) != registryToken {
		return 0, false
	}
	id := uint32(packed)
	return id, id != 0
}
func (r *hostNativeOperation) SetProfileNodeBinding(registryToken, nodeID uint32) {
	if r == nil || registryToken == 0 || nodeID == 0 {
		return
	}
	r.profileBinding.Store((uint64(registryToken) << 32) | uint64(nodeID))
}

var _ value.Callable = (*hostNativeOperation)(nil)
var _ hal.EvalContextRequired = (*hostNativeOperation)(nil)
var _ hal.ProfileNamed = (*hostNativeOperation)(nil)
var _ hal.ProfileDynamicCallable = (*hostNativeOperation)(nil)

func halHostNativeBind(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "hostnative.bind"
	if len(args) != 2 {
		return value.NewFault("%s: want 2 arguments", name)
	}
	implementationValue, ok := args[0].(*value.String)
	if !ok || implementationValue.Val == "" {
		return value.NewFault("%s: expected non-empty implementation string", name)
	}
	operationValue, ok := args[1].(*value.String)
	if !ok || operationValue.Val == "" {
		return value.NewFault("%s: expected non-empty operation string", name)
	}
	fn, err := hostnative.Resolve(implementationValue.Val, operationValue.Val)
	if err != nil {
		return value.NewFault("%s: %v", name, err)
	}
	return &hostNativeOperation{implementation: implementationValue.Val, operation: operationValue.Val, fn: fn}
}
