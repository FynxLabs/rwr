package system

import "testing"

func TestAURProvidersDeclareNestedSudo(t *testing.T) {
	providers, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"yay", "paru"} {
		p := providers[name]
		if p == nil {
			t.Fatalf("missing provider %s", name)
		}
		if p.Elevated || !p.Escalates {
			t.Errorf("%s: Elevated=%v Escalates=%v; want Elevated=false Escalates=true", name, p.Elevated, p.Escalates)
		}
	}
}

func TestAURInstallStepsDeclarePrivilegeBoundaries(t *testing.T) {
	providers, err := LoadEmbeddedProviders()
	if err != nil {
		t.Fatal(err)
	}
	p := providers["yay"]
	steps := p.Install.Steps
	if len(steps) != 3 {
		t.Fatalf("steps=%d", len(steps))
	}
	if steps[0].Elevated == nil || !*steps[0].Elevated {
		t.Fatal("pacman prerequisites must be elevated")
	}
	if steps[1].Escalates == nil || *steps[1].Escalates {
		t.Fatal("git clone must not request sudo")
	}
	if steps[2].Escalates == nil || !*steps[2].Escalates {
		t.Fatal("makepkg must use managed nested sudo")
	}
}
