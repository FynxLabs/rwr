package helpers

import (
	"testing"

	"github.com/freehold-digital/rwr/internal/types"
)

func TestCredentialDependencyValidationWithoutValues(t *testing.T) {
	c := &types.InitConfig{Credentials: []types.CredentialSpec{{Name: "token"}}}
	for _, tt := range []struct {
		data  string
		valid bool
	}{
		{`{"scripts":[{"requiresCredentials":["token"]}]}`, true},
		{`{"scripts":[{"requiresCredentials":["typo"]}]}`, false},
		{`{"scripts":[{"requiresCredentials":"token"}]}`, false},
		{`{"files":[{"content":"__RWR_CREDENTIAL_typo__"}]}`, false},
		{`{"files":[{"onCredentialUnavailable":"ignore"}]}`, false},
	} {
		if err := ValidateCredentialDependencies([]byte(tt.data), "json", c); (err == nil) != tt.valid {
			t.Fatalf("%s: %v", tt.data, err)
		}
	}
}
