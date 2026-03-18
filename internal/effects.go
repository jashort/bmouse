package internal

import "fmt"

// ─── LED Zones ───────────────────────────────────────────────────────────────

// LED zone IDs for the Basilisk V3 Pro.
const (
	ZoneAll         byte = 0x00 // all LEDs at once (scroll + logo + underglow)
	ZoneScrollWheel byte = 0x01 // scroll-wheel LED
	ZoneLogo        byte = 0x04 // logo LED
	ZoneUnderglow   byte = 0x0A // underglow light strip
)

// ZoneEach lists the individual addressable zones.
var ZoneEach = []byte{ZoneScrollWheel, ZoneLogo, ZoneUnderglow}

// ZoneByName maps human-readable names to zone IDs.
var ZoneByName = map[string]byte{
	"all":    ZoneAll,
	"scroll": ZoneScrollWheel,
	"logo":   ZoneLogo,
	"under":  ZoneUnderglow,
	"strip":  ZoneUnderglow,
}

// ZoneNameByID maps zone IDs back to their human-readable names.
var ZoneNameByID = map[byte]string{
	ZoneAll:         "all",
	ZoneScrollWheel: "scroll",
	ZoneLogo:        "logo",
	ZoneUnderglow:   "underglow",
}

// ─── Effect IDs ──────────────────────────────────────────────────────────────

const (
	EffectNone      byte = 0x00
	EffectStatic    byte = 0x01
	EffectBreathing byte = 0x02
	EffectSpectrum  byte = 0x03
	EffectReactive  byte = 0x05
)

// Command class / IDs for LED control.
const (
	ClassLED     byte = 0x0F
	CmdSetEffect byte = 0x02
	CmdGetEffect byte = 0x82
	CmdSetBright byte = 0x04
	CmdGetBright byte = 0x84
)

// ─── Storage ─────────────────────────────────────────────────────────────────

const (
	StorageVarStore byte = 0x00 // volatile / live — not persisted across power cycles
	StorageSaved    byte = 0x01 // save to the currently active hardware profile
)

// EffectName maps effect IDs to human-readable names.
var EffectName = map[byte]string{
	EffectNone:      "off",
	EffectStatic:    "static",
	EffectBreathing: "breathing",
	EffectSpectrum:  "spectrum",
	EffectReactive:  "reactive",
}

// EffectInfo holds the parsed LED effect returned by GetEffect.
type EffectInfo struct {
	EffectID   byte
	Speed      byte       // meaningful for EffectReactive (1=short,2=medium,3=long)
	ColorCount byte       // 0=random, 1=single, 2=dual
	Colors     [2][3]byte // up to two RGB triples
}

// ─── High-level effect methods ───────────────────────────────────────────────

// SetStatic sets a zone to a fixed RGB color.
// Use StorageVarStore for a temporary change, or StorageSaved to persist across power cycles.
func (d *Device) SetStatic(storage, zone, r, g, b byte) error {
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x09)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	pkt.Args[2] = EffectStatic
	pkt.Args[5] = 0x01 // color count
	pkt.Args[6] = r
	pkt.Args[7] = g
	pkt.Args[8] = b
	_, err := d.Send(pkt)
	return err
}

// SetStaticAll sets every LED zone to the same RGB color.
func (d *Device) SetStaticAll(storage, r, g, b byte) error {
	return d.SetStatic(storage, ZoneAll, r, g, b)
}

// SetBreathing sets a single-color breathing (pulsing) effect.
func (d *Device) SetBreathing(storage, zone, r, g, b byte) error {
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x09)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	pkt.Args[2] = EffectBreathing
	pkt.Args[5] = 0x01
	pkt.Args[6] = r
	pkt.Args[7] = g
	pkt.Args[8] = b
	_, err := d.Send(pkt)
	return err
}

// SetBreathingDual sets a two-color breathing effect.
func (d *Device) SetBreathingDual(storage, zone, r1, g1, b1, r2, g2, b2 byte) error {
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x0C)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	pkt.Args[2] = EffectBreathing
	pkt.Args[5] = 0x02
	pkt.Args[6] = r1
	pkt.Args[7] = g1
	pkt.Args[8] = b1
	pkt.Args[9] = r2
	pkt.Args[10] = g2
	pkt.Args[11] = b2
	_, err := d.Send(pkt)
	return err
}

// SetBreathingRandom sets a random-color breathing effect.
func (d *Device) SetBreathingRandom(storage, zone byte) error {
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x06)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	pkt.Args[2] = EffectBreathing
	pkt.Args[5] = 0x00
	_, err := d.Send(pkt)
	return err
}

// SetSpectrum sets the spectrum cycling (rainbow) effect.
func (d *Device) SetSpectrum(storage, zone byte) error {
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x06)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	pkt.Args[2] = EffectSpectrum
	_, err := d.Send(pkt)
	return err
}

// SetSpectrumAll sets spectrum cycling on every zone.
func (d *Device) SetSpectrumAll(storage byte) error {
	return d.SetSpectrum(storage, ZoneAll)
}

// SetReactive sets the reactive effect (lights up on click).
// speed: 1=short, 2=medium, 3=long.
func (d *Device) SetReactive(storage, zone, speed, r, g, b byte) error {
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x09)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	pkt.Args[2] = EffectReactive
	pkt.Args[3] = speed
	pkt.Args[5] = 0x01
	pkt.Args[6] = r
	pkt.Args[7] = g
	pkt.Args[8] = b
	_, err := d.Send(pkt)
	return err
}

// SetOff turns the LED zone off.
func (d *Device) SetOff(storage, zone byte) error {
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x06)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	pkt.Args[2] = EffectNone
	_, err := d.Send(pkt)
	return err
}

// SetOffAll turns off every LED zone.
func (d *Device) SetOffAll(storage byte) error {
	return d.SetOff(storage, ZoneAll)
}

// SetBrightness sets the brightness for a zone (0–255).
func (d *Device) SetBrightness(storage, zone, brightness byte) error {
	pkt := NewPacket(ClassLED, CmdSetBright, 0x03)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	pkt.Args[2] = brightness
	_, err := d.Send(pkt)
	return err
}

// GetBrightness reads the current brightness for a zone.
func (d *Device) GetBrightness(zone byte) (byte, error) {
	return d.getBrightnessFrom(StorageVarStore, zone)
}

// getBrightnessFrom reads brightness for a zone from a specific storage slot.
func (d *Device) getBrightnessFrom(storage, zone byte) (byte, error) {
	pkt := NewPacket(ClassLED, CmdGetBright, 0x03)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	resp, err := d.Send(pkt)
	if err != nil {
		return 0, err
	}
	return resp.Args[2], nil
}

// GetEffect reads the current LED effect for a zone from the given storage slot.
// Use StorageVarStore to read the live state.
func (d *Device) GetEffect(storage, zone byte) (EffectInfo, error) {
	pkt := NewPacket(ClassLED, CmdGetEffect, 0x0C)
	pkt.Args[0] = storage
	pkt.Args[1] = zone
	resp, err := d.Send(pkt)
	if err != nil {
		return EffectInfo{}, err
	}
	info := EffectInfo{
		EffectID:   resp.Args[2],
		Speed:      resp.Args[3],
		ColorCount: resp.Args[5],
	}
	if info.ColorCount >= 1 {
		info.Colors[0] = [3]byte{resp.Args[6], resp.Args[7], resp.Args[8]}
	}
	if info.ColorCount >= 2 {
		info.Colors[1] = [3]byte{resp.Args[9], resp.Args[10], resp.Args[11]}
	}
	return info, nil
}

// ─── Scroll Mode ─────────────────────────────────────────────────────────────

// Scroll wheel mode constants for SetScrollMode.
const (
	ScrollTactile   byte = 0x00 // clicky, notched steps (never free-spins)
	ScrollFreeSpin  byte = 0x01 // smooth, frictionless spinning
	ScrollSmartReel byte = 0x02 // auto-switch: tactile at low speed, free-spin at high speed
)

// Command IDs for scroll mode control (all class 0x02).
const (
	ClassDevice      byte = 0x02
	CmdScrollMode    byte = 0x14 // base mode: 0x00=clutch engaged, 0x01=free-spin
	CmdScrollModeSR  byte = 0x16 // smart-reel toggle: 0x00=off, 0x02=on
	CmdScrollModeSR2 byte = 0x17 // smart-reel param: 0x00=off, 0x01=on
	CmdGetScrollMode byte = 0x94
	CmdGetScrollSR   byte = 0x96
	CmdGetScrollSR2  byte = 0x97
)

// ScrollModeByName maps human-readable names to mode values.
var ScrollModeByName = map[string]byte{
	"tactile":  ScrollTactile,
	"free":     ScrollFreeSpin,
	"freespin": ScrollFreeSpin,
	"smart":    ScrollSmartReel,
	"auto":     ScrollSmartReel,
}

// SetScrollMode sets the scroll wheel mode.
//   - ScrollTactile:   pure tactile, never free-spins
//   - ScrollFreeSpin:  always free-spinning
//   - ScrollSmartReel: auto-switch based on scroll speed
func (d *Device) SetScrollMode(storage, mode byte) error {
	switch mode {
	case ScrollTactile:
		// Clutch engaged + smart-reel disabled
		if err := d.setScrollReg(storage, CmdScrollMode, 0x00); err != nil {
			return err
		}
		if err := d.setScrollReg(storage, CmdScrollModeSR, 0x00); err != nil {
			return err
		}
		return d.setScrollReg(storage, CmdScrollModeSR2, 0x00)

	case ScrollFreeSpin:
		// Clutch disengaged
		return d.setScrollReg(storage, CmdScrollMode, 0x01)

	case ScrollSmartReel:
		// Clutch engaged + smart-reel enabled
		if err := d.setScrollReg(storage, CmdScrollMode, 0x00); err != nil {
			return err
		}
		if err := d.setScrollReg(storage, CmdScrollModeSR, 0x02); err != nil {
			return err
		}
		return d.setScrollReg(storage, CmdScrollModeSR2, 0x01)

	default:
		return fmt.Errorf("unknown scroll mode 0x%02X", mode)
	}
}

// GetScrollMode reads the current scroll wheel mode.
func (d *Device) GetScrollMode() (byte, error) {
	base, err := d.getScrollReg(CmdGetScrollMode)
	if err != nil {
		return 0, err
	}
	if base == 0x01 {
		return ScrollFreeSpin, nil
	}
	// base == 0x00: check if smart-reel is on
	sr, err := d.getScrollReg(CmdGetScrollSR)
	if err != nil {
		return 0, err
	}
	if sr >= 0x01 {
		return ScrollSmartReel, nil
	}
	return ScrollTactile, nil
}

func (d *Device) setScrollReg(storage, cmd, val byte) error {
	pkt := NewPacket(ClassDevice, cmd, 0x02)
	pkt.Args[0] = storage
	pkt.Args[1] = val
	_, err := d.Send(pkt)
	return err
}

func (d *Device) getScrollReg(cmd byte) (byte, error) {
	return d.getScrollRegFrom(StorageVarStore, cmd)
}

func (d *Device) getScrollRegFrom(storage, cmd byte) (byte, error) {
	pkt := NewPacket(ClassDevice, cmd, 0x02)
	pkt.Args[0] = storage
	resp, err := d.Send(pkt)
	if err != nil {
		return 0, err
	}
	return resp.Args[1], nil
}

// getScrollModeFrom reads the scroll mode from the given storage slot.
func (d *Device) getScrollModeFrom(storage byte) (byte, error) {
	base, err := d.getScrollRegFrom(storage, CmdGetScrollMode)
	if err != nil {
		return 0, err
	}
	if base == 0x01 {
		return ScrollFreeSpin, nil
	}
	sr, err := d.getScrollRegFrom(storage, CmdGetScrollSR)
	if err != nil {
		return 0, err
	}
	if sr >= 0x01 {
		return ScrollSmartReel, nil
	}
	return ScrollTactile, nil
}
