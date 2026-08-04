package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// EncodeBlueprintDoc renders a document in a target format. CUE output is JSON-form
// CUE: valid, lossless, mechanical - idiomatic CUE is authoring work.
func EncodeBlueprintDoc(doc map[string]interface{}, format string) ([]byte, error) {
	switch format {
	case "yaml":
		return yaml.Marshal(doc)
	case "json":
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(doc); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "toml":
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(doc); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case "cue":
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "\t")
		if err := enc.Encode(doc); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	return nil, fmt.Errorf("unsupported target format %q", format)
}

// BlueprintExtension is the filename extension for a registry format.
func BlueprintExtension(format string) string {
	return "." + format
}
