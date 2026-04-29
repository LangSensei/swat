package notify

import (
	"encoding/base64"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"

	"github.com/LangSensei/swat/commander/layout"
	"github.com/LangSensei/swat/commander/operation"
)

// DesktopNotifier sends native desktop notifications via OS-specific commands.
type DesktopNotifier struct{}

// Notify sends a desktop notification. Uses osascript on macOS,
// notify-send on Linux, and PowerShell toast on Windows.
// On Windows, if opID resolves to a report.html, clicking the toast opens it.
func (d *DesktopNotifier) Notify(opID string, message string) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title "SWAT"`, message)
		return exec.Command("osascript", "-e", script).Run()
	case "linux":
		return exec.Command("notify-send", "SWAT", message).Run()
	case "windows":
		reportPath := findReportPath(opID)
		launchAttr := ""
		if reportPath != "" {
			// Convert to file:/// URL with forward slashes
			fileURL := "file:///" + strings.ReplaceAll(reportPath, `\`, "/")
			safeURL := html.EscapeString(fileURL)
			launchAttr = fmt.Sprintf(` launch="%s" activationType="protocol"`, safeURL)
		}
		safeMessage := html.EscapeString(message)
		ps := fmt.Sprintf(
			// Register AUMID so Windows does not silently swallow the toast
			`$aumid = 'SWAT'; `+
				`$regPath = "HKCU:\Software\Classes\AppUserModelId\$aumid"; `+
				`if (-not (Test-Path $regPath)) { `+
				`New-Item -Path $regPath -Force | Out-Null; `+
				`New-ItemProperty -Path $regPath -Name DisplayName -Value 'SWAT' -Force | Out-Null }; `+
				// Load WinRT types and send toast
				`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; `+
				`[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom, ContentType = WindowsRuntime] > $null; `+
				`$xml = '<toast%s><visual><binding template="ToastGeneric"><text>SWAT</text><text>%s</text></binding></visual></toast>'; `+
				`$doc = [Windows.Data.Xml.Dom.XmlDocument]::new(); `+
				`$doc.LoadXml($xml); `+
				`$toast = [Windows.UI.Notifications.ToastNotification]::new($doc); `+
				`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('SWAT').Show($toast)`,
			launchAttr, safeMessage,
		)
		encoded := encodeUTF16LEBase64(ps)
		ps51 := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
		if os.Getenv("SystemRoot") == "" {
			ps51 = "powershell"
		}
		return exec.Command(ps51, "-NoProfile", "-EncodedCommand", encoded).Run()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// findReportPath resolves the report.html path for an operation.
func findReportPath(opID string) string {
	if opID == "" {
		return ""
	}
	op, err := operation.Find(opID)
	if err != nil || op.Squad == "" {
		return ""
	}
	reportPath := filepath.Join(layout.OperationDir(op.Squad, opID), "report.html")
	if _, err := os.Stat(reportPath); err == nil {
		return reportPath
	}
	return ""
}

// encodeUTF16LEBase64 converts a string to UTF-16LE bytes and then base64-encodes it,
// suitable for PowerShell's -EncodedCommand parameter.
func encodeUTF16LEBase64(s string) string {
	runes := utf16.Encode([]rune(s))
	b := make([]byte, len(runes)*2)
	for i, r := range runes {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}
