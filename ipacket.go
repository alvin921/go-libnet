package libnet

import (
	"encoding/binary"
	"io"
)

// Packet protocol.
type IPacket interface {
	// Packet a message.
	prepareOutBuffer(buffer *OutBuffer, size int)

	// Write a packet.
	write(writer io.Writer, buffer *OutBuffer) error

	// Read a packet.
	read(reader io.Reader, buffer *InBuffer) error
}

// Create a {packet, N} protocol.
// The n means how many bytes of the packet header.
// n must is 1、2、4 or 8.
func PacketN(n int, byteOrder binary.ByteOrder) IPacket {
	return newPacketWrapper(n, byteOrder)
}

// default: 4 bytes big endian header
var iPacket = PacketN(4, binary.BigEndian)

func SetPacket(p IPacket) {
	iPacket = p
}
