package credentials

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fynxlabs/rwr/internal/types"
)

func TestAttachmentReplacementRetainsOldUntilVerified(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		uploadFails, mismatch bool
		wantDelete            bool
	}{{name: "verified", wantDelete: true}, {name: "upload fails", uploadFails: true}, {name: "verification fails", mismatch: true}} {
		t.Run(tt.name, func(t *testing.T) {
			oldCommand := providerCommand
			defer func() { providerCommand = oldCommand }()
			uploaded, verified, deleted := false, false, false
			providerCommand = func(_ context.Context, _ string, args []string, _ map[string]string, _ io.Reader) ([]byte, error) {
				switch args[0] + " " + args[1] {
				case "get item":
					if uploaded {
						return []byte(`{"id":"item-id","attachments":[{"id":"old","fileName":"private.asc"},{"id":"new","fileName":"private.asc"}]}`), nil
					}
					return []byte(`{"id":"item-id","attachments":[{"id":"old","fileName":"private.asc"}]}`), nil
				case "get attachment":
					if args[2] == "old" {
						return []byte("old-backup"), nil
					}
					verified = true
					if tt.mismatch {
						return []byte("corrupt"), nil
					}
					return []byte("new-backup"), nil
				case "create attachment":
					if tt.uploadFails {
						return nil, errors.New("upload failed")
					}
					uploaded = true
					return nil, nil
				case "delete attachment":
					if !verified {
						t.Fatal("deleted before verification")
					}
					if args[2] != "old" {
						t.Fatal("wrong attachment deleted")
					}
					deleted = true
					return nil, nil
				}
				t.Fatalf("unexpected command: %v", args)
				return nil, nil
			}
			h := &bitwardenHandle{binary: "bw", env: map[string]string{}}
			err := h.ReplaceAttachment(context.Background(), types.CredentialAttachment{Item: "key", Filename: "private.asc", Write: true}, []byte("new-backup"))
			if (err == nil) != tt.wantDelete {
				t.Fatalf("error=%v", err)
			}
			if deleted != tt.wantDelete {
				t.Fatalf("deleted=%v", deleted)
			}
		})
	}
}

func TestTaskBackupWithoutWriteProfileDoesNotResolve(t *testing.T) {
	c := &types.InitConfig{Credentials: []types.CredentialSpec{{Name: "pass"}}, CredentialAttachments: []types.CredentialAttachment{{Name: "key", Connection: "test", Write: true}}}
	r := NewResolver(c)
	defer r.Close()
	err := (TaskRunner{Resolver: r, Connection: "test"}).Run(context.Background(), types.CredentialTask{Name: "backup", Kind: "gpg-backup", Source: "key", Passphrase: "pass", WriteProfile: "backup"})
	if !IsUnavailable(err) || !strings.Contains(err.Error(), "skipped") {
		t.Fatalf("got %v", err)
	}
}
