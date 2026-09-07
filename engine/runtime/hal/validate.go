package hal

import (
	"fmt"
	"sort"
	"strings"
)

// ValidateMetadata verifies the compact HAL metadata graph. Inputs are explicit
// so invariant tests can exercise the same validator with deliberately malformed
// graphs rather than duplicating validation logic.
func ValidateMetadata(operations map[string]HostOperation) error {
	var problems []string

	identities := map[string]string{}
	for primitive, op := range operations {
		if primitive == "" {
			problems = append(problems, "empty operation key")
		}
		if primitive != op.Primitive {
			problems = append(problems, fmt.Sprintf("operation key %s disagrees with primitive %s", primitive, op.Primitive))
		}
		if op.Identity == "" {
			problems = append(problems, fmt.Sprintf("operation %s has empty HAL identity", primitive))
		} else if prior, ok := identities[op.Identity]; ok {
			problems = append(problems, fmt.Sprintf("HAL identity %s is shared by %s and %s", op.Identity, prior, primitive))
		} else {
			identities[op.Identity] = primitive
		}
		if op.Effect == "" {
			problems = append(problems, fmt.Sprintf("operation %s has empty effect", primitive))
		}
		if len(op.NezhaBindings) == 0 {
			problems = append(problems, fmt.Sprintf("operation %s has no Nezha binding", primitive))
		}
		for _, binding := range op.NezhaBindings {
			if binding.Name == "" || binding.Source == "" {
				problems = append(problems, fmt.Sprintf("operation %s has incomplete Nezha binding", primitive))
			}
		}
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("HAL metadata invariant failure:\n  %s", strings.Join(problems, "\n  "))
}

// ValidateCanonicalMetadata verifies the repository's authoritative HAL graph.
func ValidateCanonicalMetadata() error {
	return ValidateMetadata(OperationDefinitions())
}
