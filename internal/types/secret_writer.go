package types

import (
	"io"
	"strings"
)

func NewSecretWriter(target io.Writer, secrets []string) *SecretWriter {
	return &SecretWriter{target: target, secrets: secrets}
}

type SecretWriter struct {
	target  io.Writer
	secrets []string
	pending string
}

func (w *SecretWriter) Write(p []byte) (int, error) {
	w.pending += string(p)

	var output strings.Builder
	for len(w.pending) > 0 {
		matched := ""
		partial := false
		for _, secret := range w.secrets {
			if secret == "" {
				continue
			}
			if strings.HasPrefix(w.pending, secret) && len(secret) > len(matched) {
				matched = secret
			}
			if strings.HasPrefix(secret, w.pending) && len(secret) > len(w.pending) {
				partial = true
			}
		}
		if partial {
			break
		}
		if matched != "" {
			output.WriteString("[redacted]")
			w.pending = w.pending[len(matched):]
			continue
		}
		output.WriteByte(w.pending[0])
		w.pending = w.pending[1:]
	}

	_, err := io.WriteString(w.target, output.String())
	return len(p), err
}
func (w *SecretWriter) Flush() {
	value := w.pending
	for _, secret := range w.secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[redacted]")
		}
	}
	_, _ = io.WriteString(w.target, value) //nolint:errcheck // Terminal shutdown cannot recover a closed output.
	w.pending = ""
}
