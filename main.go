package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jashort/bmouse/internal"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	if cmd == "list" {
		internal.ListRazerDevices()
		return
	}

	dev, err := internal.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	defer dev.Close()

	// Parse optional --zone flag (default "all").
	zones, rest := parseZone(os.Args[2:])

	switch cmd {
	case "static":
		r, g, b := parseColor(rest, cmd)
		applyZones(dev, zones, func(z byte) error { return dev.SetStatic(z, r, g, b) })
		fmt.Printf("Static #%02X%02X%02X\n", r, g, b)

	case "breathe", "breathing":
		r, g, b := parseColor(rest, cmd)
		applyZones(dev, zones, func(z byte) error { return dev.SetBreathing(z, r, g, b) })
		fmt.Printf("Breathing #%02X%02X%02X\n", r, g, b)

	case "spectrum", "rainbow":
		applyZones(dev, zones, func(z byte) error { return dev.SetSpectrum(z) })
		fmt.Println("Spectrum cycling")

	case "wave":
		dir := byte(1)
		if len(rest) > 0 {
			if rest[0] == "right" || rest[0] == "2" {
				dir = 2
			}
		}
		applyZones(dev, zones, func(z byte) error { return dev.SetWave(z, dir) })
		fmt.Println("Wave effect")

	case "reactive":
		r, g, b := parseColor(rest, cmd)
		speed := byte(2) // medium
		applyZones(dev, zones, func(z byte) error { return dev.SetReactive(z, speed, r, g, b) })
		fmt.Printf("Reactive #%02X%02X%02X\n", r, g, b)

	case "off":
		applyZones(dev, zones, func(z byte) error { return dev.SetOff(z) })
		fmt.Println("LEDs off")

	case "brightness":
		if len(rest) == 0 {
			for _, z := range internal.ZoneEach {
				b, err := dev.GetBrightness(z)
				exitOn(err)
				fmt.Printf("Zone 0x%02X brightness: %d/255\n", z, b)
			}
		} else {
			val := mustInt(rest[0], "brightness value 0-255")
			applyZones(dev, zones, func(z byte) error { return dev.SetBrightness(z, byte(val)) })
			fmt.Printf("Brightness set to %d\n", val)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func printUsage() {
	fmt.Println(`bmouse — Razer Basilisk V3 Pro LED control (direct USB HID)

Usage:
  bmouse <command> [--zone <zone>] [args...]

Commands:
  list                      List all Razer HID devices
  static   <hex-color>      Set a solid colour          e.g. static ff0000
  breathe  <hex-color>      Breathing / pulsing effect  e.g. breathe 00ff00
  spectrum                  Rainbow spectrum cycling
  wave     [left|right]     Wave effect (default: left)
  reactive <hex-color>      Lights up on click
  off                       Turn LEDs off
  brightness [0-255]        Get or set brightness

Zones (optional --zone flag, default all):
  all      All LEDs at once
  scroll   Scroll-wheel LED
  logo     Logo LED
  under    Underglow light strip

Colour format:
  6-digit hex, with or without leading '#':  ff8800  or  #ff8800

Examples:
  bmouse static ff0000
  bmouse breathe --zone logo 00ff88
  bmouse spectrum
  bmouse off --zone scroll
  bmouse brightness 200`)
}

func parseZone(args []string) ([]byte, []string) {
	for i, a := range args {
		if a == "--zone" && i+1 < len(args) {
			name := strings.ToLower(args[i+1])
			z, ok := internal.ZoneByName[name]
			if !ok {
				fmt.Fprintf(os.Stderr, "Unknown zone %q (valid: all, scroll, logo, under)\n", name)
				os.Exit(1)
			}
			rest := append(args[:i], args[i+2:]...)
			return []byte{z}, rest
		}
	}
	return nil, args // nil means "all"
}

func parseColor(args []string, cmdName string) (r, g, b byte) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "%s requires a hex colour argument (e.g. ff0000)\n", cmdName)
		os.Exit(1)
	}
	hex := strings.TrimPrefix(args[0], "#")
	if len(hex) != 6 {
		fmt.Fprintf(os.Stderr, "Invalid colour %q — expected 6-digit hex (e.g. ff0000)\n", args[0])
		os.Exit(1)
	}
	rv, _ := strconv.ParseUint(hex[0:2], 16, 8)
	gv, _ := strconv.ParseUint(hex[2:4], 16, 8)
	bv, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return byte(rv), byte(gv), byte(bv)
}

func mustInt(s, label string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid %s: %q\n", label, s)
		os.Exit(1)
	}
	return v
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// applyZones applies fn to each zone. If zones is nil, uses ZoneAll (0x00).
func applyZones(dev *internal.Device, zones []byte, fn func(byte) error) {
	if zones == nil {
		zones = []byte{internal.ZoneAll}
	}
	for _, z := range zones {
		exitOn(fn(z))
	}
}
