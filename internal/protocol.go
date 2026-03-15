package internal

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/sstallion/go-hid"
)

// Razer USB HID wire protocol — 90-byte feature reports.
//
// Offset  Size  Description
// ------  ----  -----------
// 0       1     Status (0x00=new, 0x02=ok, 0x03=err, 0x05=unsupported)
// 1       1     Transaction ID (0x1F for newer devices)
// 2       2     Remaining packets (big-endian, usually 0)
// 4       1     Protocol type (0x00)
// 5       1     Data size (number of argument bytes used)
// 6       1     Command class
// 7       1     Command ID
// 8–87    80    Arguments
// 88      1     CRC  (XOR of bytes 2‥87)
// 89      1     Reserved (0x00)

const (
	PacketLen     = 90
	ArgsOffset    = 8
	ArgsMaxLen    = 80
	CRCOffset     = 88
	ReservedOff   = 89
	TransactionID = 0x1F
)

// Status codes returned by the device.
const (
	StatusNew         = 0x00
	StatusBusy        = 0x01
	StatusOK          = 0x02
	StatusFail        = 0x03
	StatusTimeout     = 0x04
	StatusUnsupported = 0x05
)

// Packet represents a single Razer HID feature-report command.
type Packet struct {
	Status        byte
	TransactionID byte
	Remaining     uint16 // big-endian on the wire
	ProtoType     byte
	DataSize      byte
	CommandClass  byte
	CommandID     byte
	Args          [ArgsMaxLen]byte
}

// NewPacket creates a command packet with sensible defaults.
func NewPacket(class, id, dataSize byte) Packet {
	return Packet{
		Status:        StatusNew,
		TransactionID: TransactionID,
		CommandClass:  class,
		CommandID:     id,
		DataSize:      dataSize,
	}
}

// Bytes serializes the packet to a 90-byte slice (with CRC filled in).
func (p *Packet) Bytes() [PacketLen]byte {
	var buf [PacketLen]byte
	buf[0] = p.Status
	buf[1] = p.TransactionID
	binary.BigEndian.PutUint16(buf[2:4], p.Remaining)
	buf[4] = p.ProtoType
	buf[5] = p.DataSize
	buf[6] = p.CommandClass
	buf[7] = p.CommandID
	copy(buf[ArgsOffset:ArgsOffset+ArgsMaxLen], p.Args[:])
	buf[CRCOffset] = crc(&buf)
	buf[ReservedOff] = 0x00
	return buf
}

// ParsePacket deserializes a 90-byte response into a Packet.
func ParsePacket(buf [PacketLen]byte) Packet {
	var p Packet
	p.Status = buf[0]
	p.TransactionID = buf[1]
	p.Remaining = binary.BigEndian.Uint16(buf[2:4])
	p.ProtoType = buf[4]
	p.DataSize = buf[5]
	p.CommandClass = buf[6]
	p.CommandID = buf[7]
	copy(p.Args[:], buf[ArgsOffset:ArgsOffset+ArgsMaxLen])
	return p
}

// crc computes the XOR of bytes 2‥87.
func crc(buf *[PacketLen]byte) byte {
	var c byte
	for i := 2; i < CRCOffset; i++ {
		c ^= buf[i]
	}
	return c
}

// sendCommand writes a feature report to the device and reads the response.
// The report ID used by Razer is 0x00.
func sendCommand(dev *hid.Device, pkt Packet) (Packet, error) {
	raw := pkt.Bytes()

	// Feature reports on macOS/hidapi require the report-ID as the first byte.
	// Report ID 0x00 means "default" for Razer.
	out := make([]byte, PacketLen+1)
	out[0] = 0x00 // report ID
	copy(out[1:], raw[:])

	if _, err := dev.SendFeatureReport(out); err != nil {
		return Packet{}, fmt.Errorf("send feature report: %w", err)
	}

	// Give the firmware time to process the command before reading back.
	time.Sleep(20 * time.Millisecond)

	// Read the response feature report.
	in := make([]byte, PacketLen+1)
	in[0] = 0x00 // report ID
	if _, err := dev.GetFeatureReport(in); err != nil {
		return Packet{}, fmt.Errorf("get feature report: %w", err)
	}

	var resp [PacketLen]byte
	copy(resp[:], in[1:]) // skip report-ID byte
	rp := ParsePacket(resp)

	if rp.Status == StatusFail {
		return rp, fmt.Errorf("device returned error status 0x%02X for cmd 0x%02X/0x%02X",
			rp.Status, rp.CommandClass, rp.CommandID)
	}
	if rp.Status == StatusUnsupported {
		return rp, fmt.Errorf("device does not support cmd 0x%02X/0x%02X",
			rp.CommandClass, rp.CommandID)
	}
	return rp, nil
}
