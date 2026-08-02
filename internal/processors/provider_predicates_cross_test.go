package processors

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

// The CUE schema's #ConditionRef list and repositoryPredicates are one
// contract checked from both sides: a predicate added in Go without the
// schema (providers can't use it) or in the schema without Go (conditions
// decode into nothing and every step runs) fails here.
func TestConditionRefListMatchesRepositoryPredicates(t *testing.T) {
	schema, err := os.ReadFile(filepath.Join("..", "..", "providers", "cue", "schema.cue"))
	if err != nil {
		t.Skipf("schema.cue not readable from this checkout: %v", err)
	}

	// The quoted names between #ConditionRef and the step-data comment line.
	section := string(schema)
	start := strings.Index(section, "#ConditionRef:")
	end := strings.Index(section, "// step-data fields")
	if start < 0 || end < 0 || end < start {
		t.Fatal("could not locate the #ConditionRef list in schema.cue")
	}
	var fromSchema []string
	for _, m := range regexp.MustCompile(`"([A-Za-z]+)"`).FindAllStringSubmatch(section[start:end], -1) {
		fromSchema = append(fromSchema, m[1])
	}
	// The final two entries before the marker are step-data fields, not predicates.
	dataFields := map[string]bool{"ResetSettings": true, "URL": true}
	var schemaPredicates []string
	for _, name := range fromSchema {
		if !dataFields[name] {
			schemaPredicates = append(schemaPredicates, name)
		}
	}
	sort.Strings(schemaPredicates)

	var fromGo []string
	for name := range repositoryPredicates(types.Repository{}, nil) {
		fromGo = append(fromGo, name)
	}
	sort.Strings(fromGo)

	if strings.Join(schemaPredicates, ",") != strings.Join(fromGo, ",") {
		t.Errorf("schema #ConditionRef predicates != repositoryPredicates keys\n schema: %v\n go:     %v", schemaPredicates, fromGo)
	}
}
