package ops

import (
	"encoding/json"
	"fmt"
	"strings"
)

type manifestDoc struct {
	Permissions []string `json:"permissions"`
	Examples    []struct {
		Command string `json:"command"`
		Safety  string `json:"safety"`
	} `json:"examples"`
	SafetyNotes []string `json:"safety_notes"`
}

// ValidateManifest checks that toolkit command references map to operations.
func ValidateManifest(data []byte, operations []Operation) error {
	var doc manifestDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	known := map[string]Operation{}
	for _, op := range operations {
		known[op.Tool+" "+op.Name] = op
	}
	for _, permission := range doc.Permissions {
		if err := validateCommandRef(permission, known, doc.SafetyNotes); err != nil {
			return err
		}
	}
	for _, ex := range doc.Examples {
		if err := validateCommandRef(ex.Command, known, append(doc.SafetyNotes, ex.Safety)); err != nil {
			return err
		}
	}
	return nil
}

func validateCommandRef(command string, known map[string]Operation, safety []string) error {
	parts := strings.Fields(command)
	if len(parts) < 3 || parts[0] != "toolkit" {
		return nil
	}
	key := parts[1] + " " + parts[2]
	op, ok := known[key]
	if !ok {
		return fmt.Errorf("unknown operation in manifest command %q", command)
	}
	if !op.ReadOnly && len(safety) == 0 {
		return fmt.Errorf("write-capable operation %s.%s requires manifest safety notes", op.Tool, op.Name)
	}
	return nil
}
