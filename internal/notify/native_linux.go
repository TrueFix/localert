package notify

import (
	"context"
	"fmt"
	"html"
	"image/png"
	"os"

	"github.com/godbus/dbus/v5"
)

func (Native) Send(ctx context.Context, n Notification, iconPath string) error {
	conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("connect to desktop session: %w", err)
	}
	defer conn.Close()
	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	// Send pixels over D-Bus: the desktop notification daemon cannot read a
	// container's private filesystem, even when its session bus is shared.
	hints := map[string]dbus.Variant{"urgency": dbus.MakeVariant(byte(1))}
	file, err := os.Open(iconPath)
	if err != nil {
		return err
	}
	img, err := png.Decode(file)
	file.Close()
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	pixels := make([]byte, 0, bounds.Dx()*bounds.Dy()*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a > 0 {
				r, g, b = r*65535/a, g*65535/a, b*65535/a
			}
			pixels = append(pixels, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
		}
	}
	hints["image-data"] = dbus.MakeVariant(struct {
		Width     int32
		Height    int32
		Rowstride int32
		Alpha     bool
		Bits      int32
		Channels  int32
		Data      []byte
	}{int32(bounds.Dx()), int32(bounds.Dy()), int32(bounds.Dx() * 4), true, 8, 4, pixels})
	var id uint32
	err = obj.CallWithContext(ctx, "org.freedesktop.Notifications.Notify", 0,
		"Localert", uint32(0), "", n.Title, html.EscapeString(n.Message),
		[]string{}, hints, int32(-1),
	).Store(&id)
	if err != nil {
		return fmt.Errorf("desktop notification service: %w", err)
	}
	return nil
}
