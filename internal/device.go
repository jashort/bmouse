package internal

import (
	"fmt"

	"github.com/sstallion/go-hid"
)

// VendorID Razer vendor ID (all Razer peripherals).
const VendorID = 0x1532

// Known product IDs for Basilisk V3 variants.
const (
	PIDBasiliskV3        = 0x0099
	PIDBasiliskV3Pro     = 0x00AA
	PIDBasiliskV3ProWL   = 0x00AB
	PIDBasiliskV3XHSpeed = 0x00AC
	PIDBasiliskV3XBT     = 0x00AD
)

// knownDevices is the single source of truth for supported PIDs and their
// human-readable names. Open iterates this slice in order; to add support
// for a new variant, add one entry here (and a PID constant above).
var knownDevices = []struct {
	pid  uint16
	name string
}{
	{PIDBasiliskV3, "Basilisk V3"},
	{PIDBasiliskV3Pro, "Basilisk V3 Pro (Wired)"},
	{PIDBasiliskV3ProWL, "Basilisk V3 Pro (Wireless)"},
	{PIDBasiliskV3XHSpeed, "Basilisk V3 X HyperSpeed"},
	{PIDBasiliskV3XBT, "Basilisk V3 X BT"},
}

// Device wraps an opened HID handle to a Razer mouse.
type Device struct {
	hid       *hid.Device
	ProductID uint16
	Name      string
}

// Open finds and opens the first Razer Basilisk V3 (any variant).
func Open() (*Device, error) {
	if err := hid.Init(); err != nil {
		return nil, fmt.Errorf("hid init: %w", err)
	}

	for _, kd := range knownDevices {
		d, err := openPID(kd.pid)
		if err == nil {
			return d, nil
		}
	}

	return nil, fmt.Errorf("no Razer Basilisk V3 found (is it plugged in?)")
}

// openPID tries to open interface 0 for a specific Razer product ID.
// Interface 0 (the mouse HID input interface) is the one that handles
// Razer protocol feature reports on the Basilisk V3 Pro.
func openPID(pid uint16) (*Device, error) {
	var targetPath string

	if err := hid.Enumerate(VendorID, pid, func(info *hid.DeviceInfo) error {
		if info.InterfaceNbr == 0 && targetPath == "" {
			targetPath = info.Path
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("enumerate pid 0x%04X: %w", pid, err)
	}

	if targetPath == "" {
		return nil, fmt.Errorf("pid 0x%04X not found", pid)
	}

	h, err := hid.OpenPath(targetPath)
	if err != nil {
		return nil, fmt.Errorf("open 0x%04X: %w", pid, err)
	}

	name := fmt.Sprintf("Razer 0x%04X", pid)
	for _, kd := range knownDevices {
		if kd.pid == pid {
			name = kd.name
			break
		}
	}
	return &Device{hid: h, ProductID: pid, Name: name}, nil
}

// Close releases the HID handle.
func (d *Device) Close() {
	if d.hid != nil {
		d.hid.Close()
	}
	hid.Exit()
}

// Send sends a protocol command and returns the response.
func (d *Device) Send(pkt Packet) (Packet, error) {
	return sendCommand(d.hid, pkt)
}

// ListRazerDevices returns every Razer HID interface visible on the system.
// Useful for finding the correct product ID.
func ListRazerDevices() ([]hid.DeviceInfo, error) {
	if err := hid.Init(); err != nil {
		return nil, fmt.Errorf("hid init: %w", err)
	}
	defer hid.Exit()

	var devices []hid.DeviceInfo
	if err := hid.Enumerate(VendorID, 0x0000, func(info *hid.DeviceInfo) error {
		devices = append(devices, *info)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("enumerate: %w", err)
	}
	return devices, nil
}
