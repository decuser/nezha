package primitives

import (
	"fmt"
	"sort"
	"strings"

	"nezha/engine/runtime/hal"
)

// ValidateDefinitions verifies the primitive-name catalog against the canonical
// HAL operation set. Inputs are explicit so invariant assurance can mutate a
// copy and prove the same validator rejects drift.
func ValidateDefinitions(definitions map[string]bool, operations map[string]hal.HostOperation) error {
	var problems []string
	for name := range definitions {
		if name == "" {
			problems = append(problems, "empty primitive identity")
		}
	}
	for primitive := range operations {
		if !definitions[primitive] {
			problems = append(problems, fmt.Sprintf("HAL primitive %s has no architectural definition", primitive))
		}
	}
	for name := range definitions {
		if _, isHAL := operations[name]; !isHAL && !runtimePrimitives[name] {
			problems = append(problems, fmt.Sprintf("runtime primitive %s is not runtime or HAL-defined", name))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("primitive catalog invariant failure:\n  %s", strings.Join(problems, "\n  "))
}

func ValidateCanonicalDefinitions() error {
	return ValidateDefinitions(Definitions(), hal.OperationDefinitions())
}
