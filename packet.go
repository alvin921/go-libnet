package libnet

import (
	"encoding/binary"
	"io"
)

type ByteOrder binary.ByteOrder

var (
	BigEndian    = ByteOrder(binary.BigEndian)
	LittleEndian = ByteOrder(binary.LittleEndian)

	packet1BE = newPacketWrapper(1, BigEndian)
	packet1LE = newPacketWrapper(1, LittleEndian)
	packet2BE = newPacketWrapper(2, BigEndian)
	packet2LE = newPacketWrapper(2, LittleEndian)
	packet4BE = newPacketWrapper(4, BigEndian)
	packet4LE = newPacketWrapper(4, LittleEndian)
	packet8BE = newPacketWrapper(8, BigEndian)
	packet8LE = newPacketWrapper(8, LittleEndian)
)

// The packet spliting protocol like Erlang's {packet, N}.
// Each packet has a fix length packet header to present packet length.
type packetWrapper struct {
	n             int
	bo            binary.ByteOrder
	encodeHead    func([]byte)
	decodeHead    func([]byte) int
	MaxPacketSize int
}

func newPacketWrapper(n int, byteOrder binary.ByteOrder) *packetWrapper {
	protocol := &packetWrapper{
		n:  n,
		bo: byteOrder,
	}

	switch n {
	case 1:
		protocol.encodeHead = func(buffer []byte) {
			buffer[0] = byte(len(buffer) - n)
		}
		protocol.decodeHead = func(buffer []byte) int {
			return int(buffer[0])
		}
	case 2:
		protocol.encodeHead = func(buffer []byte) {
			byteOrder.PutUint16(buffer, uint16(len(buffer)-n))
		}
		protocol.decodeHead = func(buffer []byte) int {
			return int(byteOrder.Uint16(buffer))
		}
	case 4:
		protocol.encodeHead = func(buffer []byte) {
			byteOrder.PutUint32(buffer, uint32(len(buffer)-n))
		}
		protocol.decodeHead = func(buffer []byte) int {
			return int(byteOrder.Uint32(buffer))
		}
	case 8:
		protocol.encodeHead = func(buffer []byte) {
			byteOrder.PutUint64(buffer, uint64(len(buffer)-n))
		}
		protocol.decodeHead = func(buffer []byte) int {
			return int(byteOrder.Uint64(buffer))
		}
	default:
		panic("unsupported packet head size")
	}

	return protocol
}

func (p *packetWrapper) prepareOutBuffer(buffer *OutBuffer, size int) {
	buffer.Prepare(size)
	buffer.Data = buffer.Data[:p.n]
}

func (p *packetWrapper) write(writer io.Writer, packet *OutBuffer) error {
	if p.MaxPacketSize > 0 && len(packet.Data) > p.MaxPacketSize {
		return PacketTooLargeError
	}
	p.encodeHead(packet.Data)
	if _, err := writer.Write(packet.Data); err != nil {
		return err
	}
	return nil
}

func (p *packetWrapper) read(reader io.Reader, buffer *InBuffer) error {
	// head
	buffer.Prepare(p.n)
	if _, err := io.ReadFull(reader, buffer.Data); err != nil {
		return err
	}
	size := p.decodeHead(buffer.Data)

	if p.MaxPacketSize > 0 && size > p.MaxPacketSize {
		return PacketTooLargeError
	}
	// body
	buffer.Prepare(size)
	if size == 0 {
		return nil
	}

	if _, err := io.ReadFull(reader, buffer.Data); err != nil {
		return err
	}

	return nil
}
