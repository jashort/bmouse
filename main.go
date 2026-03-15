package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jashort/bmouse/internal"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		printUsage()
		return fmt.Errorf("no command specified")
	}

	cmd := os.Args[1]

	if cmd == "list" {
		devices, err := internal.ListRazerDevices()
		if err != nil {
			return err
		}
		if len(devices) == 0 {
			fmt.Println("(none found – is a Razer device plugged in?)")
			return nil
		}
		fmt.Println("Razer HID devices:")
		for _, info := range devices {
			name := info.ProductStr
			if name == "" {
				name = "(unnamed)"
			}
			fmt.Printf("  PID=0x%04X  %-30s  UsagePage=0x%04X  Usage=0x%04X  Interface=%d  Path=%s\n",
				info.ProductID, name, info.UsagePage, info.Usage, info.InterfaceNbr, info.Path)
		}
		return nil
	}

	dev, err := internal.Open()
	if err != nil {
		return err
	}
	defer dev.Close()

	// Parse optional --zone flag (default "all").
	zones, rest, err := parseZone(os.Args[2:])
	if err != nil {
		return err
	}

	switch cmd {
	case "static":
		r, g, b, err := parseColor(rest, cmd)
		if err != nil {
			return err
		}
		if err := applyZones(zones, func(z byte) error { return dev.SetStatic(z, r, g, b) }); err != nil {
			return err
		}
		fmt.Printf("Static #%02X%02X%02X\n", r, g, b)

	case "breathe", "breathing":
		r, g, b, err := parseColor(rest, cmd)
		if err != nil {
			return err
		}
		if err := applyZones(zones, func(z byte) error { return dev.SetBreathing(z, r, g, b) }); err != nil {
			return err
		}
		fmt.Printf("Breathing #%02X%02X%02X\n", r, g, b)

	case "breathe-dual", "breathing-dual":
		if len(rest) < 2 {
			return fmt.Errorf("breathe-dual requires two hex colours (e.g. breathe-dual ff0000 0000ff)")
		}
		r1, g1, b1, err := parseColor(rest, cmd)
		if err != nil {
			return err
		}
		r2, g2, b2, err := parseColor(rest[1:], cmd)
		if err != nil {
			return err
		}
		if err := applyZones(zones, func(z byte) error {
			return dev.SetBreathingDual(z, r1, g1, b1, r2, g2, b2)
		}); err != nil {
			return err
		}
		fmt.Printf("Breathing dual #%02X%02X%02X / #%02X%02X%02X\n", r1, g1, b1, r2, g2, b2)

	case "spectrum", "rainbow":
		if err := applyZones(zones, func(z byte) error { return dev.SetSpectrum(z) }); err != nil {
			return err
		}
		fmt.Println("Spectrum cycling")

	case "wave":
		dir := byte(1)
		if len(rest) > 0 {
			if rest[0] == "right" || rest[0] == "2" {
				dir = 2
			}
		}
		if err := applyZones(zones, func(z byte) error { return dev.SetWave(z, dir) }); err != nil {
			return err
		}
		fmt.Println("Wave effect")

	case "reactive":
		speedStr, rest, err := parseFlag(rest, "speed")
		if err != nil {
			return err
		}
		speed := byte(2) // medium default
		if speedStr != "" {
			v, err := parseInt(speedStr, "speed (1=short, 2=medium, 3=long)")
			if err != nil {
				return err
			}
			if v < 1 || v > 3 {
				return fmt.Errorf("--speed must be 1 (short), 2 (medium), or 3 (long)")
			}
			speed = byte(v)
		}
		r, g, b, err := parseColor(rest, cmd)
		if err != nil {
			return err
		}
		if err := applyZones(zones, func(z byte) error { return dev.SetReactive(z, speed, r, g, b) }); err != nil {
			return err
		}
		fmt.Printf("Reactive #%02X%02X%02X speed=%d\n", r, g, b, speed)

	case "off":
		if err := applyZones(zones, func(z byte) error { return dev.SetOff(z) }); err != nil {
			return err
		}
		fmt.Println("LEDs off")

	case "brightness":
		if len(rest) == 0 {
			for _, z := range internal.ZoneEach {
				b, err := dev.GetBrightness(z)
				if err != nil {
					return err
				}
				fmt.Printf("Zone 0x%02X brightness: %d/255\n", z, b)
			}
		} else {
			val, err := parseInt(rest[0], "brightness value 0-255")
			if err != nil {
				return err
			}
			if err := applyZones(zones, func(z byte) error { return dev.SetBrightness(z, byte(val)) }); err != nil {
				return err
			}
			fmt.Printf("Brightness set to %d\n", val)
		}

	case "scroll":
		if len(rest) == 0 {
			mode, err := dev.GetScrollMode()
			if err != nil {
				return err
			}
			names := map[byte]string{
				internal.ScrollTactile:   "tactile",
				internal.ScrollFreeSpin:  "free-spin",
				internal.ScrollSmartReel: "smart-reel (auto)",
			}
			name := names[mode]
			if name == "" {
				name = fmt.Sprintf("unknown (0x%02X)", mode)
			}
			fmt.Printf("Scroll mode: %s\n", name)
		} else {
			modeName := strings.ToLower(rest[0])
			mode, ok := internal.ScrollModeByName[modeName]
			if !ok {
				return fmt.Errorf("unknown scroll mode %q (valid: tactile, free, smart)", modeName)
			}
			if err := dev.SetScrollMode(mode); err != nil {
				return err
			}
			fmt.Printf("Scroll mode set to %s\n", modeName)
		}

	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", cmd)
	}

	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func printUsage() {
	fmt.Println(`bmouse — Razer Basilisk V3 Pro LED control (direct USB HID)

Usage:
  bmouse <command> [--zone <zone>] [args...]

Commands:
  list                           List all Razer HID devices
  static       <hex-color>       Set a solid colour           e.g. static ff0000
  breathe      <hex-color>       Single-colour breathing      e.g. breathe 00ff00
  breathe-dual <color1> <color2> Two-colour breathing         e.g. breathe-dual ff0000 0000ff
  spectrum                       Rainbow spectrum cycling
  wave         [left|right]      Wave effect (default: left)
  reactive     <hex-color>       Lights up on click
               [--speed 1-3]       1=short  2=medium(default)  3=long
  off                            Turn LEDs off
  brightness   [0-255]           Get or set brightness
  scroll       [mode]            Get or set scroll wheel mode
                                   Modes: tactile, free, smart

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
  bmouse breathe-dual ff0000 0000ff
  bmouse reactive ff0000 --speed 1
  bmouse spectrum
  bmouse off --zone scroll
  bmouse brightness 200
  bmouse scroll tactile
  bmouse scroll free
  bmouse scroll smart`)
}

func parseZone(args []string) ([]byte, []string, error) {
	for i, a := range args {
		if a == "--zone" && i+1 < len(args) {
			name := strings.ToLower(args[i+1])
			z, ok := internal.ZoneByName[name]
			if !ok {
				return nil, nil, fmt.Errorf("unknown zone %q (valid: all, scroll, logo, under)", name)
			}
			rest := append(args[:i], args[i+2:]...)
			return []byte{z}, rest, nil
		}
	}
	return nil, args, nil // nil zones means "all"
}

// parseFlag removes --name value from args and returns the value and remaining
// args. Returns ("", args, nil) if the flag is not present.
func parseFlag(args []string, name string) (string, []string, error) {
	flag := "--" + name
	for i, a := range args {
		if a == flag {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("%s requires a value", flag)
			}
			rest := append(args[:i], args[i+2:]...)
			return args[i+1], rest, nil
		}
	}
	return "", args, nil
}

func parseColor(args []string, cmdName string) (r, g, b byte, err error) {
	if len(args) == 0 {
		return 0, 0, 0, fmt.Errorf("%s requires a hex colour argument (e.g. ff0000)", cmdName)
	}
	hex := strings.TrimPrefix(args[0], "#")
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid colour %q — expected 6-digit hex (e.g. ff0000)", args[0])
	}
	rv, _ := strconv.ParseUint(hex[0:2], 16, 8)
	gv, _ := strconv.ParseUint(hex[2:4], 16, 8)
	bv, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return byte(rv), byte(gv), byte(bv), nil
}

func parseInt(s, label string) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q", label, s)
	}
	return v, nil
}

// applyZones applies fn to each zone. If zones is nil, uses ZoneAll (0x00).
func applyZones(zones []byte, fn func(byte) error) error {
	if zones == nil {
		zones = []byte{internal.ZoneAll}
	}
	for _, z := range zones {
		if err := fn(z); err != nil {
			return err
		}
	}
	return nil
}
