package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"

	qrcode "github.com/skip2/go-qrcode"
)

// localIP returns the first private IPv4 on an up, non-loopback adapter —
// the address a phone on the same Wi-Fi can reach.
func localIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipn.IP.To4()
			if ip != nil && ip.IsPrivate() {
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no private IPv4 adapter found")
}

// printQR renders the LAN URL as an ASCII QR block in the console.
func printQR(url string) {
	q, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		fmt.Println("scan manually:", url)
		return
	}
	fmt.Println(q.ToSmallString(false))
	fmt.Println("  scan with a phone on the same Wi-Fi:", url)
}

// WriteQRSVG dumps the same code as an SVG file, for embedding in docs or a VI.
func WriteQRSVG(url, path string) error {
	q, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		return err
	}
	bm := q.Bitmap()
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" shape-rendering="crispEdges" viewBox="0 0 %d %d">`, len(bm), len(bm))
	svg += `<rect width="100%" height="100%" fill="#fff"/>`
	for y, row := range bm {
		for x, on := range row {
			if on {
				svg += fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1" fill="#000"/>`, x, y)
			}
		}
	}
	return os.WriteFile(path, []byte(svg+"</svg>"), 0o644)
}

// openBrowser launches the desktop default browser at the local URL.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Println("open manually:", url)
	}
}
