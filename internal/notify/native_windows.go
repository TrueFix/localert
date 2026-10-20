package notify

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
)

func (Native) Send(ctx context.Context, n Notification, iconPath string) error {
	escape := func(s string) string {
		var b strings.Builder
		_ = xml.EscapeText(&b, []byte(s))
		return b.String()
	}
	audio := `<audio silent="true"/>`
	if n.Sound {
		audio = `<audio src="ms-winsoundevent:Notification.Default"/>`
	}
	toast := `<toast><visual><binding template="ToastGeneric"><text>` + escape(n.Title) +
		`</text><text>` + escape(n.Message) + `</text><image placement="appLogoOverride" src="` +
		escape(iconPath) + `"/></binding></visual>` + audio + `</toast>`
	script := `$ErrorActionPreference = 'Stop'
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] > $null
$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml([Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($env:LOCALERT_TOAST)))
$app = 'HKCU:\Software\Classes\AppUserModelId\Localert'
if (!(Test-Path $app)) {
  New-Item -Path $app -Force > $null
  New-ItemProperty -Path $app -Name DisplayName -Value 'Localert' -PropertyType String -Force > $null
  New-ItemProperty -Path $app -Name ShowInSettings -Value 1 -PropertyType DWord -Force > $null
}
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Localert').Show($toast)`
	encoded := utf16.Encode([]rune(script))
	bytes := make([]byte, len(encoded)*2)
	for i, v := range encoded {
		binary.LittleEndian.PutUint16(bytes[i*2:], v)
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(bytes))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Env = append(os.Environ(), "LOCALERT_TOAST="+base64.StdEncoding.EncodeToString([]byte(toast)))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Windows notification: %w: %s", err, out)
	}
	return nil
}
