package libnet

import "sync"

// The channel type. Used to maintain a group of session.
// Normally used for broadcast classify purpose.
type Channel struct {
	sync.RWMutex
	sessions    map[uint64]channelSession
	broadcaster *Broadcaster

	// channel state
	State interface{}
}

type channelSession struct {
	*Session
	KickCallback func()
}

// Create a channel instance.
func NewChannel() *Channel {
	channel := &Channel{
		sessions: make(map[uint64]channelSession),
	}
	channel.broadcaster = NewBroadcaster(iPacket, channel.Fetch)
	return channel
}

// Broadcast to channel. The message only encoded once
// so the performance is better than send message one by one.
func (channel *Channel) Broadcast(encoder Encoder) ([]BroadcastWork, error) {
	return channel.broadcaster.Broadcast(encoder)
}

// How mush sessions in this channel.
func (channel *Channel) Len() int {
	channel.RLock()
	defer channel.RUnlock()

	return len(channel.sessions)
}

// Join the channel. The kickCallback will called when the session kick out from the channel.
func (channel *Channel) Join(session *Session, kickCallback func()) {
	channel.Lock()
	defer channel.Unlock()

	session.AddCloseCallback(channel, func() {
		channel.Exit(session)
	})
	channel.sessions[session.Id()] = channelSession{session, kickCallback}
}

// Exit the channel.
func (channel *Channel) Exit(session *Session) {
	channel.Lock()
	defer channel.Unlock()

	session.RemoveCloseCallback(channel)
	delete(channel.sessions, session.Id())
}

// Kick out a session from the channel.
func (channel *Channel) Kick(session *Session) {
	channel.Lock()
	defer channel.Unlock()

	sessionId := session.Id()
	if session, exists := channel.sessions[sessionId]; exists {
		delete(channel.sessions, sessionId)
		if session.KickCallback != nil {
			session.KickCallback()
		}
	}
}

// fetch the sessions. NOTE: Invoke Kick() or Exit() in fetch callback will dead lock.
func (channel *Channel) Fetch(callback func(*Session)) {
	channel.RLock()
	defer channel.RUnlock()

	for _, sesssion := range channel.sessions {
		callback(sesssion.Session)
	}
}
