package internal

import (
	"encoding/binary"
	"testing"
)

// TestNewPacket verifies that NewPacket sets the documented defaults and leaves
// all Args bytes zero.
func TestNewPacket(t *testing.T) {
	p := NewPacket(0x0F, 0x02, 0x09)

	if p.Status != StatusNew {
		t.Errorf("Status = 0x%02X, want 0x%02X (StatusNew)", p.Status, StatusNew)
	}
	if p.TransactionID != TransactionID {
		t.Errorf("TransactionID = 0x%02X, want 0x%02X", p.TransactionID, TransactionID)
	}
	if p.CommandClass != 0x0F {
		t.Errorf("CommandClass = 0x%02X, want 0x0F", p.CommandClass)
	}
	if p.CommandID != 0x02 {
		t.Errorf("CommandID = 0x%02X, want 0x02", p.CommandID)
	}
	if p.DataSize != 0x09 {
		t.Errorf("DataSize = 0x%02X, want 0x09", p.DataSize)
	}
	for i, b := range p.Args {
		if b != 0 {
			t.Errorf("Args[%d] = 0x%02X, want 0x00", i, b)
		}
	}
}

// TestPacketBytes_Layout verifies that Bytes() places each field at the correct
// wire offset defined by the Razer HID protocol.
func TestPacketBytes_Layout(t *testing.T) {
	p := NewPacket(0x0F, 0x02, 0x09)
	p.Remaining = 0x1234
	p.Args[0] = 0xAB
	p.Args[ArgsMaxLen-1] = 0xCD

	buf := p.Bytes()

	cases := []struct {
		name string
		got  byte
		want byte
	}{
		{"buf[0] Status", buf[0], StatusNew},
		{"buf[1] TransactionID", buf[1], TransactionID},
		{"buf[4] ProtoType", buf[4], 0x00},
		{"buf[5] DataSize", buf[5], 0x09},
		{"buf[6] CommandClass", buf[6], 0x0F},
		{"buf[7] CommandID", buf[7], 0x02},
		{"buf[ArgsOffset] Args[0]", buf[ArgsOffset], 0xAB},
		{"buf[ArgsOffset+79] Args[79]", buf[ArgsOffset+ArgsMaxLen-1], 0xCD},
		{"buf[ReservedOff]", buf[ReservedOff], 0x00},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = 0x%02X, want 0x%02X", tc.name, tc.got, tc.want)
		}
	}

	if got := binary.BigEndian.Uint16(buf[2:4]); got != 0x1234 {
		t.Errorf("buf[2:4] Remaining = 0x%04X, want 0x1234", got)
	}
}

// TestPacketCRC verifies the CRC field via both Bytes() (integration) and the
// unexported crc() function directly (unit).
func TestPacketCRC(t *testing.T) {
	// --- integration: CRC produced by Bytes() must equal XOR of bytes 2‥87 ---
	p := NewPacket(0x0F, 0x02, 0x09)
	p.Args[0] = 0xDE
	p.Args[1] = 0xAD
	buf := p.Bytes()

	var want byte
	for i := 2; i < CRCOffset; i++ {
		want ^= buf[i]
	}
	if buf[CRCOffset] != want {
		t.Errorf("Bytes() CRC = 0x%02X, want 0x%02X", buf[CRCOffset], want)
	}

	// --- unit: crc() directly ---

	// All-zero buffer → CRC is 0x00.
	var zeroBuf [PacketLen]byte
	if got := crc(&zeroBuf); got != 0x00 {
		t.Errorf("crc(all-zero) = 0x%02X, want 0x00", got)
	}

	// Single non-zero byte inside the window → CRC equals that byte.
	var oneBuf [PacketLen]byte
	oneBuf[5] = 0x42
	if got := crc(&oneBuf); got != 0x42 {
		t.Errorf("crc(single 0x42 at index 5) = 0x%02X, want 0x42", got)
	}

	// Two bytes inside the window → CRC equals their XOR.
	var twoBuf [PacketLen]byte
	twoBuf[5] = 0xF0
	twoBuf[10] = 0x0F
	if got := crc(&twoBuf); got != 0xFF {
		t.Errorf("crc(0xF0 ^ 0x0F) = 0x%02X, want 0xFF", got)
	}

	// Bytes outside the window (indices 0, 1, 88, 89) are not included.
	var outBuf [PacketLen]byte
	outBuf[0] = 0xFF
	outBuf[1] = 0xFF
	outBuf[88] = 0xFF
	outBuf[89] = 0xFF
	if got := crc(&outBuf); got != 0x00 {
		t.Errorf("crc(bytes outside window) = 0x%02X, want 0x00", got)
	}
}

// TestParsePacket verifies that ParsePacket correctly deserializes a manually
// constructed raw buffer into a Packet.
func TestParsePacket(t *testing.T) {
	var raw [PacketLen]byte
	raw[0] = StatusOK
	raw[1] = 0x1F // TransactionID
	raw[2] = 0x00
	raw[3] = 0x00 // Remaining = 0
	raw[4] = 0x00 // ProtoType
	raw[5] = 0x09 // DataSize
	raw[6] = 0x0F // CommandClass
	raw[7] = 0x02 // CommandID
	raw[8] = 0xAB // Args[0]
	raw[9] = 0xCD // Args[1]

	p := ParsePacket(raw)

	cases := []struct {
		name string
		got  byte
		want byte
	}{
		{"Status", p.Status, StatusOK},
		{"TransactionID", p.TransactionID, 0x1F},
		{"DataSize", p.DataSize, 0x09},
		{"CommandClass", p.CommandClass, 0x0F},
		{"CommandID", p.CommandID, 0x02},
		{"Args[0]", p.Args[0], 0xAB},
		{"Args[1]", p.Args[1], 0xCD},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("ParsePacket %s = 0x%02X, want 0x%02X", tc.name, tc.got, tc.want)
		}
	}

	if p.Remaining != 0x0000 {
		t.Errorf("ParsePacket Remaining = 0x%04X, want 0x0000", p.Remaining)
	}
}

// TestPacketRoundTrip checks that a Packet serialised with Bytes() and then
// deserialize with ParsePacket() recovers all original field values exactly.
func TestPacketRoundTrip(t *testing.T) {
	p := NewPacket(0x0F, 0x02, 0x09)
	p.Args[0] = StorageVarStore
	p.Args[1] = ZoneScrollWheel
	p.Args[2] = EffectStatic
	p.Args[5] = 0x01
	p.Args[6] = 0xFF // r
	p.Args[7] = 0x80 // g
	p.Args[8] = 0x00 // b

	got := ParsePacket(p.Bytes())

	if got.Status != p.Status {
		t.Errorf("Status: got 0x%02X, want 0x%02X", got.Status, p.Status)
	}
	if got.TransactionID != p.TransactionID {
		t.Errorf("TransactionID: got 0x%02X, want 0x%02X", got.TransactionID, p.TransactionID)
	}
	if got.Remaining != p.Remaining {
		t.Errorf("Remaining: got 0x%04X, want 0x%04X", got.Remaining, p.Remaining)
	}
	if got.DataSize != p.DataSize {
		t.Errorf("DataSize: got 0x%02X, want 0x%02X", got.DataSize, p.DataSize)
	}
	if got.CommandClass != p.CommandClass {
		t.Errorf("CommandClass: got 0x%02X, want 0x%02X", got.CommandClass, p.CommandClass)
	}
	if got.CommandID != p.CommandID {
		t.Errorf("CommandID: got 0x%02X, want 0x%02X", got.CommandID, p.CommandID)
	}
	if got.Args != p.Args {
		t.Errorf("Args mismatch\n got:  %v\n want: %v", got.Args, p.Args)
	}
}

// TestPacketRoundTrip_ZeroPacket ensures that a zero-value Packet serialises
// and deserializes without panicking and with all fields preserved.
func TestPacketRoundTrip_ZeroPacket(t *testing.T) {
	var p Packet
	got := ParsePacket(p.Bytes())

	if got.Status != 0 {
		t.Errorf("Status: got 0x%02X, want 0x00", got.Status)
	}
	if got.TransactionID != 0 {
		t.Errorf("TransactionID: got 0x%02X, want 0x00", got.TransactionID)
	}
	if got.Args != p.Args {
		t.Errorf("Args mismatch for zero packet")
	}
}
