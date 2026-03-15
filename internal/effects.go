package internal

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

// ─── Effect IDs ──────────────────────────────────────────────────────────────

const (
	EffectNone      byte = 0x00
	EffectStatic    byte = 0x01
	EffectBreathing byte = 0x02
	EffectSpectrum  byte = 0x03
	EffectWave      byte = 0x04
	EffectReactive  byte = 0x05
)

// Command class / IDs for LED control.
const (
	ClassLED     byte = 0x0F
	CmdSetEffect byte = 0x02
	CmdSetBright byte = 0x04
	CmdGetBright byte = 0x84
)

// ─── Storage ─────────────────────────────────────────────────────────────────

const (
	StorageVarStore byte = 0x00 // volatile / live
)

// ─── Internal helpers ────────────────────────────────────────────────────────

// ensureBrightness sets brightness to 255 for the given zone.
// The V3 Pro requires brightness to be set before an effect is visible.
func (d *Device) ensureBrightness(zone byte) error {
	return d.SetBrightness(zone, 0xFF)
}

// ─── High-level effect methods ───────────────────────────────────────────────

// SetStatic sets a zone to a fixed RGB colour.
func (d *Device) SetStatic(zone, r, g, b byte) error {
	if err := d.ensureBrightness(zone); err != nil {
		return err
	}
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x09)
	pkt.Args[0] = StorageVarStore
	pkt.Args[1] = zone
	pkt.Args[2] = EffectStatic
	pkt.Args[5] = 0x01 // colour count
	pkt.Args[6] = r
	pkt.Args[7] = g
	pkt.Args[8] = b
	_, err := d.Send(pkt)
	return err
}

// SetStaticAll sets every LED zone to the same RGB colour.
func (d *Device) SetStaticAll(r, g, b byte) error {
	return d.SetStatic(ZoneAll, r, g, b)
}

// SetBreathing sets a single-colour breathing (pulsing) effect.
func (d *Device) SetBreathing(zone, r, g, b byte) error {
	if err := d.ensureBrightness(zone); err != nil {
		return err
	}
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x09)
	pkt.Args[0] = StorageVarStore
	pkt.Args[1] = zone
	pkt.Args[2] = EffectBreathing
	pkt.Args[5] = 0x01
	pkt.Args[6] = r
	pkt.Args[7] = g
	pkt.Args[8] = b
	_, err := d.Send(pkt)
	return err
}

// SetBreathingDual sets a two-colour breathing effect.
func (d *Device) SetBreathingDual(zone, r1, g1, b1, r2, g2, b2 byte) error {
	if err := d.ensureBrightness(zone); err != nil {
		return err
	}
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x0C)
	pkt.Args[0] = StorageVarStore
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

// SetBreathingRandom sets a random-colour breathing effect.
func (d *Device) SetBreathingRandom(zone byte) error {
	if err := d.ensureBrightness(zone); err != nil {
		return err
	}
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x06)
	pkt.Args[0] = StorageVarStore
	pkt.Args[1] = zone
	pkt.Args[2] = EffectBreathing
	pkt.Args[5] = 0x00
	_, err := d.Send(pkt)
	return err
}

// SetSpectrum sets the spectrum cycling (rainbow) effect.
func (d *Device) SetSpectrum(zone byte) error {
	if err := d.ensureBrightness(zone); err != nil {
		return err
	}
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x06)
	pkt.Args[0] = StorageVarStore
	pkt.Args[1] = zone
	pkt.Args[2] = EffectSpectrum
	_, err := d.Send(pkt)
	return err
}

// SetSpectrumAll sets spectrum cycling on every zone.
func (d *Device) SetSpectrumAll() error {
	return d.SetSpectrum(ZoneAll)
}

// SetWave sets the wave effect. direction: 1 = left→right, 2 = right→left.
func (d *Device) SetWave(zone, direction byte) error {
	if err := d.ensureBrightness(zone); err != nil {
		return err
	}
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x06)
	pkt.Args[0] = StorageVarStore
	pkt.Args[1] = zone
	pkt.Args[2] = EffectWave
	pkt.Args[3] = direction
	_, err := d.Send(pkt)
	return err
}

// SetReactive sets the reactive effect (lights up on click).
// speed: 1=short, 2=medium, 3=long.
func (d *Device) SetReactive(zone, speed, r, g, b byte) error {
	if err := d.ensureBrightness(zone); err != nil {
		return err
	}
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x09)
	pkt.Args[0] = StorageVarStore
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
func (d *Device) SetOff(zone byte) error {
	pkt := NewPacket(ClassLED, CmdSetEffect, 0x06)
	pkt.Args[0] = StorageVarStore
	pkt.Args[1] = zone
	pkt.Args[2] = EffectNone
	_, err := d.Send(pkt)
	return err
}

// SetOffAll turns off every LED zone.
func (d *Device) SetOffAll() error {
	return d.SetOff(ZoneAll)
}

// SetBrightness sets the brightness for a zone (0–255).
func (d *Device) SetBrightness(zone, brightness byte) error {
	pkt := NewPacket(ClassLED, CmdSetBright, 0x03)
	pkt.Args[0] = StorageVarStore
	pkt.Args[1] = zone
	pkt.Args[2] = brightness
	_, err := d.Send(pkt)
	return err
}

// GetBrightness reads the current brightness for a zone.
func (d *Device) GetBrightness(zone byte) (byte, error) {
	pkt := NewPacket(ClassLED, CmdGetBright, 0x03)
	pkt.Args[0] = StorageVarStore
	pkt.Args[1] = zone
	resp, err := d.Send(pkt)
	if err != nil {
		return 0, err
	}
	return resp.Args[2], nil
}
