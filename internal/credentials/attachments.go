package credentials

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fynxlabs/rwr/internal/types"
)

type vaultAttachment struct {
	ID       string `json:"id"`
	Filename string `json:"fileName"`
}
type vaultItem struct {
	ID          string            `json:"id"`
	Attachments []vaultAttachment `json:"attachments"`
}

func (h *bitwardenHandle) item(ctx context.Context, item string) (vaultItem, error) {
	raw, err := h.call(ctx, []string{"get", "item", item})
	var v vaultItem
	if err != nil {
		return v, err
	}
	if json.Unmarshal(raw, &v) != nil || v.ID == "" {
		return v, fmt.Errorf("invalid Bitwarden item response")
	}
	return v, nil
}
func matchingAttachment(v vaultItem, filename string) (vaultAttachment, error) {
	var match vaultAttachment
	for _, a := range v.Attachments {
		if a.Filename == filename {
			if match.ID != "" {
				return match, fmt.Errorf("multiple attachments have the configured filename; select a unique binding")
			}
			match = a
		}
	}
	if match.ID == "" {
		return match, &Unavailable{Missing}
	}
	return match, nil
}
func (h *bitwardenHandle) attachmentBytes(ctx context.Context, item, id string) ([]byte, error) {
	return h.call(ctx, []string{"get", "attachment", id, "--itemid", item, "--raw"})
}
func (h *bitwardenHandle) ReadAttachment(ctx context.Context, b types.CredentialAttachment) ([]byte, error) {
	item, err := h.item(ctx, b.Item)
	if err != nil {
		return nil, err
	}
	attachment, err := matchingAttachment(item, b.Filename)
	if err != nil {
		return nil, err
	}
	return h.attachmentBytes(ctx, item.ID, attachment.ID)
}
func (h *bitwardenHandle) ReplaceAttachment(ctx context.Context, b types.CredentialAttachment, content []byte) error {
	if !b.Write {
		return fmt.Errorf("attachment binding does not authorize writes")
	}
	before, err := h.item(ctx, b.Item)
	if err != nil {
		return err
	}
	var old []string
	for _, a := range before.Attachments {
		if a.Filename == b.Filename {
			old = append(old, a.ID)
		}
	}
	if len(old) > 1 {
		return fmt.Errorf("ambiguous existing attachments; refusing replacement")
	}
	if len(old) == 1 {
		existing, e := h.attachmentBytes(ctx, before.ID, old[0])
		if e != nil {
			return e
		}
		if bytes.Equal(existing, content) {
			return nil
		}
	}
	dir, err := os.MkdirTemp("", "rwr-attachment-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir) //nolint:errcheck
	path := filepath.Join(dir, b.Filename)
	if filepath.Base(b.Filename) != b.Filename {
		return fmt.Errorf("invalid attachment filename")
	}
	if err := os.WriteFile(path, content, 0600); err != nil {
		return err
	}
	if _, err := h.call(ctx, []string{"create", "attachment", "--file", path, "--itemid", before.ID}); err != nil {
		return err
	}
	after, err := h.item(ctx, before.ID)
	if err != nil {
		return err
	}
	var newID string
	for _, a := range after.Attachments {
		if a.Filename != b.Filename {
			continue
		}
		isOld := false
		for _, id := range old {
			if id == a.ID {
				isOld = true
			}
		}
		if !isOld {
			if newID != "" {
				return fmt.Errorf("attachment changed concurrently; old backup retained")
			}
			newID = a.ID
		}
	}
	if newID == "" {
		return fmt.Errorf("new attachment was not found; old backup retained")
	}
	uploaded, err := h.attachmentBytes(ctx, before.ID, newID)
	if err != nil {
		return err
	}
	if !bytes.Equal(uploaded, content) {
		return fmt.Errorf("uploaded attachment verification failed; old backup retained")
	}
	for _, id := range old {
		if _, err := h.call(ctx, []string{"delete", "attachment", id, "--itemid", before.ID}); err != nil {
			return fmt.Errorf("new backup verified but old attachment cleanup failed")
		}
	}
	return nil
}
