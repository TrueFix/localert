package notify

import (
	"context"
	"fmt"
	"html"

	"github.com/godbus/dbus/v5"
)

func (Native) Send(ctx context.Context, n Notification, iconPath string) error {
	conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("connect to desktop session: %w", err)
	}
	defer conn.Close()
	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	var id uint32
	err = obj.CallWithContext(ctx, "org.freedesktop.Notifications.Notify", 0,
		"Localert", uint32(0), iconPath, n.Title, html.EscapeString(n.Message),
		[]string{}, map[string]dbus.Variant{"urgency": dbus.MakeVariant(byte(1))}, int32(-1),
	).Store(&id)
	if err != nil {
		return fmt.Errorf("desktop notification service: %w", err)
	}
	return nil
}
