package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
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

func run() (err error) {
	if len(os.Args) < 2 {
		printUsage()
		return fmt.Errorf("no command specified")
	}

	cmd := os.Args[1]

	if cmd == "list" {
		return runList()
	}

	if cmd == "version" {
		return runVersion()
	}

	// Build a FlagSet shared by all subcommands.
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.Usage = printUsage

	zoneName := fs.String("zone", "all", "LED zone: all, scroll, logo, under")
	speed := fs.Int("speed", 2, "Reactive speed: 1=short 2=medium 3=long")

	if err := fs.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	zones, err := resolveZones(*zoneName)
	if err != nil {
		return err
	}

	args := fs.Args() // positional arguments after flags

	dev, err := internal.Open()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, dev.Close())
	}()

	switch cmd {
	case "status":
		return runStatus(dev)

	case "static":
		r, g, b, err := parseColor(args, cmd)
		if err != nil {
			return err
		}
		if err := applyZones(zones, func(z byte) error { return dev.SetStatic(internal.StorageSaved, z, r, g, b) }); err != nil {
			return err
		}
		fmt.Printf("Static #%02X%02X%02X (zone: %s)\n", r, g, b, *zoneName)

	case "breathe", "breathing":
		r, g, b, err := parseColor(args, cmd)
		if err != nil {
			return err
		}
		if err := applyZones(zones, func(z byte) error { return dev.SetBreathing(internal.StorageSaved, z, r, g, b) }); err != nil {
			return err
		}
		fmt.Printf("Breathing #%02X%02X%02X (zone: %s)\n", r, g, b, *zoneName)

	case "breathe-dual", "breathing-dual":
		if len(args) < 2 {
			return fmt.Errorf("breathe-dual requires two hex colors (e.g. breathe-dual ff0000 0000ff)")
		}
		r1, g1, b1, err := parseColor(args, cmd)
		if err != nil {
			return err
		}
		r2, g2, b2, err := parseColor(args[1:], cmd)
		if err != nil {
			return err
		}
		if err := applyZones(zones, func(z byte) error {
			return dev.SetBreathingDual(internal.StorageSaved, z, r1, g1, b1, r2, g2, b2)
		}); err != nil {
			return err
		}
		fmt.Printf("Breathing dual #%02X%02X%02X / #%02X%02X%02X (zone: %s)\n", r1, g1, b1, r2, g2, b2, *zoneName)

	case "spectrum", "rainbow":
		if err := applyZones(zones, func(z byte) error { return dev.SetSpectrum(internal.StorageSaved, z) }); err != nil {
			return err
		}
		fmt.Printf("Spectrum cycling (zone: %s)\n", *zoneName)

	case "reactive":
		if *speed < 1 || *speed > 3 {
			return fmt.Errorf("--speed must be 1 (short), 2 (medium), or 3 (long)")
		}
		r, g, b, err := parseColor(args, cmd)
		if err != nil {
			return err
		}
		if err := applyZones(zones, func(z byte) error {
			return dev.SetReactive(internal.StorageSaved, z, byte(*speed), r, g, b)
		}); err != nil {
			return err
		}
		fmt.Printf("Reactive #%02X%02X%02X speed=%d (zone: %s)\n", r, g, b, *speed, *zoneName)

	case "off":
		if err := applyZones(zones, func(z byte) error { return dev.SetOff(internal.StorageSaved, z) }); err != nil {
			return err
		}
		fmt.Printf("LEDs off (zone: %s)\n", *zoneName)

	case "brightness":
		if len(args) == 0 {
			for _, z := range internal.ZoneEach {
				b, err := dev.GetBrightness(z)
				if err != nil {
					return err
				}
				fmt.Printf("Zone 0x%02X brightness: %d/255\n", z, b)
			}
		} else {
			val, err := parseInt(args[0], "brightness value 0-255")
			if err != nil {
				return err
			}
			if err := applyZones(zones, func(z byte) error { return dev.SetBrightness(internal.StorageSaved, z, byte(val)) }); err != nil {
				return err
			}
			fmt.Printf("Brightness set to %d (zone: %s)\n", val, *zoneName)
		}

	case "scroll":
		if len(args) == 0 {
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
			modeName := strings.ToLower(args[0])
			mode, ok := internal.ScrollModeByName[modeName]
			if !ok {
				return fmt.Errorf("unknown scroll mode %q (valid: tactile, free, smart)", modeName)
			}
			if err := dev.SetScrollMode(internal.StorageSaved, mode); err != nil {
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

// ─── subcommand handlers ──────────────────────────────────────────────────────

func runVersion() error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("version information not available (binary was not built with module support)")
		return nil
	}

	commit := "(unknown)"
	date := "(unknown)"
	modified := false

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
		case "vcs.time":
			date = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}

	if modified {
		commit += " (modified)"
	}

	version := info.Main.Version
	if version == "" || version == "(devel)" {
		version = "dev"
	}

	fmt.Printf("bmouse %s\n", version)
	fmt.Printf("  commit: %s\n", commit)
	fmt.Printf("  built:  %s\n", date)
	fmt.Printf("  go:     %s\n", info.GoVersion)
	return nil
}

func runList() error {
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

func runStatus(dev *internal.Device) error {
	// Read current live settings from volatile storage (0x00).
	// This always reflects what is currently active on the hardware,
	// regardless of which profile slot the hardware button is pointing at.
	scrollModeNames := map[byte]string{
		internal.ScrollTactile:   "tactile",
		internal.ScrollFreeSpin:  "free-spin",
		internal.ScrollSmartReel: "smart-reel",
	}

	fmt.Println("Current settings (active profile):")
	fmt.Println()
	for _, zone := range internal.ZoneEach {
		brightness, err := dev.GetBrightness(zone)
		if err != nil {
			return fmt.Errorf("zone 0x%02X brightness: %w", zone, err)
		}
		effect, err := dev.GetEffect(internal.StorageVarStore, zone)
		if err != nil {
			// Non-fatal: show brightness only if effect read fails.
			fmt.Fprintf(os.Stderr, "warning: zone 0x%02X effect: %v\n", zone, err)
		}
		zoneName := internal.ZoneNameByID[zone]
		fmt.Printf("  %-10s  brightness=%3d  %s\n", zoneName, brightness, formatEffect(effect))
	}

	scrollMode, err := dev.GetScrollMode()
	if err != nil {
		return fmt.Errorf("scroll mode: %w", err)
	}
	scrollName := scrollModeNames[scrollMode]
	if scrollName == "" {
		scrollName = fmt.Sprintf("unknown(0x%02X)", scrollMode)
	}
	fmt.Printf("  %-10s  %s\n", "scroll", scrollName)
	return nil
}

// formatEffect converts an EffectInfo into a human-readable string.
func formatEffect(e internal.EffectInfo) string {
	switch e.EffectID {
	case internal.EffectStatic:
		c := e.Colors[0]
		return fmt.Sprintf("static #%02X%02X%02X", c[0], c[1], c[2])
	case internal.EffectBreathing:
		switch e.ColorCount {
		case 1:
			c := e.Colors[0]
			return fmt.Sprintf("breathing #%02X%02X%02X", c[0], c[1], c[2])
		case 2:
			c1, c2 := e.Colors[0], e.Colors[1]
			return fmt.Sprintf("breathing-dual #%02X%02X%02X / #%02X%02X%02X",
				c1[0], c1[1], c1[2], c2[0], c2[1], c2[2])
		default:
			return "breathing (random)"
		}
	case internal.EffectSpectrum:
		return "spectrum"
	case internal.EffectReactive:
		c := e.Colors[0]
		return fmt.Sprintf("reactive #%02X%02X%02X speed=%d", c[0], c[1], c[2], e.Speed)
	case internal.EffectNone:
		return "off"
	default:
		return fmt.Sprintf("unknown(0x%02X)", e.EffectID)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func printUsage() {
	fmt.Println(`bmouse — Razer Basilisk V3 Pro LED control (direct USB HID)

Usage:
  bmouse <command> [--zone <zone>] [args...]

Commands:
  status                          Show current active-profile settings
  static       <hex-color>       Set a solid color           e.g. static ff0000
  breathe      <hex-color>       Single-color breathing      e.g. breathe 00ff00
  breathe-dual <color1> <color2> Two-color breathing         e.g. breathe-dual ff0000 0000ff
  spectrum                       Rainbow spectrum cycling
  reactive     <hex-color>       Lights up on click
               [--speed 1-3]       1=short  2=medium(default)  3=long
  off                            Turn LEDs off
  brightness   [0-255]           Get or set brightness
  scroll       [mode]            Get or set scroll wheel mode
                                   Modes: tactile, free, smart
  list                           List all Razer HID devices
  version                        Print version and build info

Flags:
  --zone <zone>      LED zone: all (default), scroll, logo, under
  --speed <1-3>      Reactive duration: 1=short  2=medium  3=long

color format:
  6-digit hex, with or without leading '#':  ff8800  or  #ff8800

Examples:
  bmouse static ff0000
  bmouse breathe --zone logo 00ff88
  bmouse breathe-dual ff0000 0000ff
  bmouse reactive ff0000 --speed 1
  bmouse spectrum
  bmouse off --zone scroll
  bmouse brightness 200
  bmouse brightness                      (show current brightness per zone)
  bmouse scroll tactile
  bmouse scroll free
  bmouse scroll smart
  bmouse status                          (show all current settings)`)
}

// resolveZones converts a zone name to a slice of zone bytes.
func resolveZones(name string) ([]byte, error) {
	z, ok := internal.ZoneByName[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("unknown zone %q (valid: all, scroll, logo, under)", name)
	}
	return []byte{z}, nil
}

func parseColor(args []string, cmdName string) (r, g, b byte, err error) {
	if len(args) == 0 {
		return 0, 0, 0, fmt.Errorf("%s requires a hex color argument (e.g. ff0000)", cmdName)
	}
	hex := strings.TrimPrefix(args[0], "#")
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid color %q — expected 6-digit hex (e.g. ff0000)", args[0])
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
