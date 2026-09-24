package omarchy

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/freehold-digital/rwr/internal/types"
)

func hookPayload(h types.OmarchyHook) string {
	return ".local/state/rwr/omarchy/hooks/" + h.Event + "/" + h.Name + ".script"
}
func hookOutcome(h types.OmarchyHook) string {
	return ".local/state/rwr/omarchy/hooks/" + h.Event + "/" + h.Name + ".outcome"
}
func (c *Client) hookWrapper(h types.OmarchyHook) []byte {
	return []byte("#!/bin/bash\n# RWR hook: preserve arguments and record execution outcome.\numask 077\nbash " + shellQuote(filepath.Join(c.Home, hookPayload(h))) + " \"$@\"\nrwr_hook_status=$?\nprintf '%s %s\\n' \"${RWR_OMARCHY_HOOK_RUN:-manual}\" \"$rwr_hook_status\" > " + shellQuote(filepath.Join(c.Home, hookOutcome(h))) + "\nexit \"$rwr_hook_status\"\n")
}
func (c *Client) activateTheme(name string) error {
	nonce := fmt.Sprint(time.Now().UnixNano())
	if err := c.Run(types.Command{Exec: "omarchy", Args: []string{"theme", "set", name}, Variables: map[string]string{"RWR_OMARCHY_HOOK_RUN": nonce}}); err != nil {
		return err
	}
	for _, h := range c.hooks {
		raw, err := c.read(hookOutcome(h))
		if err != nil || trim(raw) != nonce+" 0" {
			return fmt.Errorf("theme hook %s did not report successful execution", h.Name)
		}
	}
	return nil
}
