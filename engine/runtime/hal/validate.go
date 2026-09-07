package hal

import (
	"fmt"
	"sort"
	"strings"
)

var validContexts = map[string]bool{
	"runtime.io": true,
}

var validEffects = map[string]bool{
	"bounded call": true,
}

var validBlocking = map[string]bool{
	"may block": true,
}

var validLifetimes = map[string]bool{
	"stateless": true,
}

var validOptionality = map[string]bool{
	"constitutive": true,
}

var validErrorContracts = map[string]bool{
	"fault": true,
}

// ValidateMetadata verifies the internal HAL metadata graph. Inputs are explicit
// so invariant tests can exercise the same validator with deliberately malformed
// graphs rather than duplicating validation logic.
func ValidateMetadata(operations map[string]HostOperation) error {
	var problems []string

	identities := map[string]string{}
	for primitive, op := range operations {
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
		if op.Authority != op.Identity {
			problems = append(problems, fmt.Sprintf("operation %s authority %s does not match canonical identity %s", primitive, op.Authority, op.Identity))
		}
		if len(op.Context) == 0 {
			problems = append(problems, fmt.Sprintf("operation %s has no context", primitive))
		}
		seenContext := map[string]bool{}
		for _, context := range op.Context {
			if !validContexts[context] {
				problems = append(problems, fmt.Sprintf("operation %s has unknown context %s", primitive, context))
			}
			if seenContext[context] {
				problems = append(problems, fmt.Sprintf("operation %s repeats context %s", primitive, context))
			}
			seenContext[context] = true
		}
		if !validEffects[op.Effect] {
			problems = append(problems, fmt.Sprintf("operation %s has unknown effect %q", primitive, op.Effect))
		}
		if !validBlocking[op.Blocking] {
			problems = append(problems, fmt.Sprintf("operation %s has unknown blocking class %q", primitive, op.Blocking))
		}
		if !validLifetimes[op.Lifetime] {
			problems = append(problems, fmt.Sprintf("operation %s has unknown lifetime %q", primitive, op.Lifetime))
		}
		if !validOptionality[op.Optionality] {
			problems = append(problems, fmt.Sprintf("operation %s has unknown optionality %q", primitive, op.Optionality))
		}
		if !validErrorContracts[op.ErrorContract] {
			problems = append(problems, fmt.Sprintf("operation %s has unknown error contract %q", primitive, op.ErrorContract))
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
