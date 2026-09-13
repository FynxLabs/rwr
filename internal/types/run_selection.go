package types

import (
	"fmt"
	"strings"
)

const BlueprintTypeCredentials = "credentials"

// RunSelection keeps all, explicit, and empty selections distinct. It also
// carries the acquisition boundary, which consumers must obey independently
// of whether a credentials blueprint was discovered.
type RunSelection struct {
	Order         []string
	Explicit      bool
	DenyProviders bool
	Bootstrap     bool
}

func NormalizeProcessor(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "repository" {
		name = BlueprintTypeRepositories
	}
	switch name {
	case BlueprintTypeCredentials, BlueprintTypeBootstrap, BlueprintTypePackages,
		BlueprintTypeRepositories, BlueprintTypeFiles, BlueprintTypeServices,
		BlueprintTypeUsers, BlueprintTypeGit, BlueprintTypeScripts, BlueprintTypeSSHKeys,
		BlueprintTypeFonts, BlueprintTypeConfiguration:
		return name, nil
	}
	return "", fmt.Errorf("unknown processor %q", name)
}

func SelectRun(c *InitConfig, requested, defaultOrder []string) (RunSelection, error) {
	s := RunSelection{Explicit: requested != nil, Order: []string{}}
	excluded := map[string]bool{}
	cli := map[string]bool{}
	for _, group := range []struct {
		names  []string
		target map[string]bool
	}{{c.Init.Except, excluded}, {c.Variables.Flags.Except, cli}} {
		for _, raw := range group.names {
			for _, part := range strings.Split(raw, ",") {
				name, err := NormalizeProcessor(part)
				if err != nil {
					return s, err
				}
				group.target[name] = true
			}
		}
	}
	if s.Explicit {
		excluded = map[string]bool{}
	}
	for name := range cli {
		excluded[name] = true
	}
	s.DenyProviders = excluded[BlueprintTypeCredentials]
	s.Bootstrap = !excluded[BlueprintTypeBootstrap] && requested == nil
	order := defaultOrder
	if s.Explicit {
		order = requested
	}
	for _, raw := range order {
		name, err := NormalizeProcessor(raw)
		if err != nil {
			return s, err
		}
		if excluded[name] {
			if s.Explicit && cli[name] {
				return s, fmt.Errorf("processor %q was both requested and excluded", name)
			}
			continue
		}
		if name == BlueprintTypeCredentials && !s.Explicit && c.CredentialPolicy.Setup != "ordered" {
			continue
		}
		s.Order = append(s.Order, name)
	}
	return s, nil
}

type CredentialPolicy struct {
	Setup         string `mapstructure:"setup,omitempty" yaml:"setup,omitempty" json:"setup,omitempty" toml:"setup,omitempty"`
	OnUnavailable string `mapstructure:"onUnavailable,omitempty" yaml:"onUnavailable,omitempty" json:"onUnavailable,omitempty" toml:"onUnavailable,omitempty"`
}

func (p CredentialPolicy) Validate() error {
	if p.Setup != "" && p.Setup != "explicit" && p.Setup != "ordered" {
		return fmt.Errorf("credentialPolicy.setup must be explicit or ordered")
	}
	if p.OnUnavailable != "" && p.OnUnavailable != "skip" && p.OnUnavailable != "fail" {
		return fmt.Errorf("credentialPolicy.onUnavailable must be skip or fail")
	}
	return nil
}
