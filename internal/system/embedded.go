package system

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/fynxlabs/rwr/internal/types"
)

//go:embed definitions/*.cue
var embeddedProviderCUE embed.FS

// LoadEmbeddedProviders evaluates the embedded CUE provider definitions -
// schema, family templates, one file per provider - and returns them keyed
// by provider name. Each call returns fresh values: callers own what they
// get, exactly as they did with the per-call decode this replaced.
func LoadEmbeddedProviders() (map[string]*types.Provider, error) {
	overlay := map[string]load.Source{}
	err := fs.WalkDir(embeddedProviderCUE, "definitions", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".cue") {
			return walkErr
		}
		data, readErr := embeddedProviderCUE.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		overlay["/providers/"+filepath.Base(path)] = load.FromBytes(data)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading embedded provider definitions: %w", err)
	}

	instances := load.Instances([]string{"/providers"}, &load.Config{Dir: "/", Overlay: overlay})
	if len(instances) == 0 || instances[0].Err != nil {
		return nil, fmt.Errorf("loading embedded provider definitions: %w", instances[0].Err)
	}
	value := cuecontext.New().BuildInstance(instances[0])
	if value.Err() != nil {
		return nil, fmt.Errorf("evaluating embedded provider definitions: %w", value.Err())
	}
	providersValue := value.LookupPath(cue.ParsePath("providers"))
	if providersValue.Err() != nil {
		return nil, fmt.Errorf("embedded definitions carry no providers struct: %w", providersValue.Err())
	}

	// The schema closes #Provider and unifies every entry against it, so
	// evaluation already enforced the shape.
	providers := map[string]*types.Provider{}
	fields, err := providersValue.Fields()
	if err != nil {
		return nil, err
	}
	for fields.Next() {
		var provider types.Provider
		if err := fields.Value().Decode(&provider); err != nil {
			return nil, fmt.Errorf("provider %s: %w", fields.Selector(), err)
		}
		providers[provider.Name] = &provider
	}
	log.Debugf("LoadEmbeddedProviders: evaluated %d providers from embedded CUE", len(providers))
	return providers, nil
}
