package libnet

// Broadcaster.
type Broadcaster struct {
	iPacket IPacket
	fetcher func(func(*Session))
}

// Broadcast work.
type BroadcastWork struct {
	Session *Session
	AsyncWork
}

// Create a broadcaster.
func NewBroadcaster(iPacket IPacket, fetcher func(func(*Session))) *Broadcaster {
	return &Broadcaster{
		iPacket: iPacket,
		fetcher: fetcher,
	}
}

// Broadcast to sessions. The message only encoded once
// so the performance is better than send message one by one.
func (b *Broadcaster) Broadcast(encoder Encoder) ([]BroadcastWork, error) {
	buffer := newOutBuffer()
	b.iPacket.prepareOutBuffer(buffer, 1024)
	if err := encoder(buffer); err != nil {
		buffer.free()
		return nil, err
	}
	buffer.isBroadcast = true
	works := make([]BroadcastWork, 0, 10)
	b.fetcher(func(session *Session) {
		// 防止给发送者广播消息
		if session.Id() != buffer.id {
			buffer.broadcastUse()
			works = append(works, BroadcastWork{
				session,
				session.AsyncSendBuffer(buffer),
			})
		}
	})
	return works, nil
}
