package types

import "testing"

func TestRunSelection(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		requested, init, cli  []string
		ordered               bool
		want                  []string
		deny, bootstrap, fail bool
	}{
		{name: "explicit default", want: []string{"scripts"}, bootstrap: true},
		{name: "ordered", ordered: true, want: []string{"credentials", "scripts"}, bootstrap: true},
		{name: "init excludes", init: []string{"credentials"}, deny: true, want: []string{"scripts"}, bootstrap: true},
		{name: "explicit override", requested: []string{"credentials"}, init: []string{"credentials"}, want: []string{"credentials"}},
		{name: "conflict", requested: []string{"credentials"}, cli: []string{"credentials"}, fail: true},
		{name: "unknown", cli: []string{"typo"}, fail: true},
		{name: "empty", requested: []string{}, want: []string{}},
		{name: "all excluded", cli: []string{"credentials,scripts,bootstrap"}, want: []string{}, deny: true},
		{name: "alias absent", cli: []string{"repository"}, want: []string{"scripts"}, bootstrap: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := &InitConfig{}
			c.Init.Except = tt.init
			c.Variables.Flags.Except = tt.cli
			if tt.ordered {
				c.CredentialPolicy.Setup = "ordered"
			}
			got, err := SelectRun(c, tt.requested, []string{"credentials", "scripts"})
			if (err != nil) != tt.fail {
				t.Fatalf("error = %v", err)
			}
			if err != nil {
				return
			}
			if len(got.Order) != len(tt.want) {
				t.Fatalf("order=%v want=%v", got.Order, tt.want)
			}
			for i, p := range tt.want {
				if got.Order[i] != p {
					t.Fatalf("order=%v", got.Order)
				}
			}
			if got.DenyProviders != tt.deny || got.Bootstrap != tt.bootstrap {
				t.Fatalf("selection=%+v", got)
			}
		})
	}
}
