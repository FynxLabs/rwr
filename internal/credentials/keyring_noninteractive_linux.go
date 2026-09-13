//go:build linux

package credentials

import (
	"context"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

// GetNoninteractive reads only already-unlocked items. It never calls Unlock
// or Prompt, and never autolaunches a desktop session bus.
func (osKeyring) GetNoninteractive(name string) (string, error) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		return "", ErrKeyringUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return "", ErrKeyringUnavailable
	}
	defer conn.Close() //nolint:errcheck
	const service = "org.freedesktop.secrets"
	object := conn.Object(service, "/org/freedesktop/secrets")
	var unlocked, locked []dbus.ObjectPath
	if err := object.CallWithContext(ctx, "org.freedesktop.Secret.Service.SearchItems", 0, map[string]string{"service": keyringService, "username": name}).Store(&unlocked, &locked); err != nil {
		return "", ErrKeyringUnavailable
	}
	if len(unlocked) != 1 {
		return "", ErrKeyringNotFound
	}
	var algorithm dbus.Variant
	var session dbus.ObjectPath
	if err := object.CallWithContext(ctx, "org.freedesktop.Secret.Service.OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&algorithm, &session); err != nil {
		return "", ErrKeyringUnavailable
	}
	defer conn.Object(service, session).CallWithContext(ctx, "org.freedesktop.Secret.Session.Close", 0)
	var secret struct {
		Session     dbus.ObjectPath
		Parameters  []byte
		Value       []byte
		ContentType string
	}
	if err := conn.Object(service, unlocked[0]).CallWithContext(ctx, "org.freedesktop.Secret.Item.GetSecret", 0, session).Store(&secret); err != nil {
		return "", ErrKeyringUnavailable
	}
	return string(secret.Value), nil
}
