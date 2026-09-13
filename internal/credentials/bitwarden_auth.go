package credentials

import (
	"fmt"
	"net/url"
)

func validateBitwardenServer(server string) error {
	parsed, err := url.Parse(server)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("bitwarden server must be an HTTPS URL with a host and no embedded credentials or fragment")
	}
	return nil
}
