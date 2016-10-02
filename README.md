# go-libnet
golang libnet package

[TOC]

## IPacket

每个packet头是N（1/2/4/8）字节，表明其后packet数据长度，从而接收方可以直接交给应用层整个packet。
```golang
func PacketN(n int, byteOrder binary.ByteOrder) IPacket
```

比如`PacketN(4, binary.BigEndian)`表示packet头是4字节BigEndian

IPacket接口如下，仅对Session开放，不对用户开放


```golang
// Packet protocol.
type PacketProtocol interface {
    // Packet a message.
    prepareOutBuffer(buffer *OutBuffer, size int)
    // Write a packet.
    write(writer io.Writer, buffer *OutBuffer) error
    // Read a packet.
    read(reader io.Reader, buffer *InBuffer) error
}
```



## Session

Session是对net.Conn的封装，提供更高级别更简易的操作

- Dial/DialTimeout

```golang
Dial(network, address string) (*Session, error)
DialTimeout(network, address string, timeout time.Duration) (*Session, error)
```

`network`: "tcp", "udp"

`address`: "ip:port", 比如“192.168.1.100:8888"

创建连接，并启动线程处理异步消息发送

- Close

```
func (session *Session) Close()
```

关闭连接，并退出所有子线程

- Send

```golang
func (session *Session) Send(encoder Encoder) error
```

发送packet，这是一个通用接口，默认提供如下几种协议封装接口: Bytes/String/Json/Gob/Xml



- AsyncSend

```golang
func (session *Session)AsyncSend(encoder Encoder) AsyncWork
```

异步发送，范例如下：

```golang
aw := sessioin.AsyncSend(libnet.Json(cmd))
err := aw.Wait()
if err == nil {
  // 异步发送成功
} else if err == SendToClosedError {
  // 异步发送失败，连接关闭
} else if err == AsyncSendTimeoutError {
  // 异步发送失败，超时(5s)
} else {
  // 其他错误	
}
```



- ProcessOnce/Process

原型如下：


```golang
type IPacketProc interface {
	Handle(data []byte, session *Session) error
}
func (session *Session)ProcessOnce(handler IPacketProc) error
func (session *Session)Process(handler IPacketProc) error
```

ProcessOnce阻塞方式读取并处理一个packet

Process阻塞方式读取并处理packet

- Heartbeat

```golang
func (session *Session)HeartBeat(timeout time.Duration, encoder Encoder)
```

定时发送心跳包，比如

```golang
session.HeartBeat(20*time.Second, libnet.String("heartbeat"))
```

如果心跳时间有变，直接继续调用，会自动调整心跳时间

如果停止发送心跳，则调用

```golang
session.HeartBeat(0, nil)
```



- Echo客户端例子：

```golang
import "github.com/alvin921/go-libnet"

func (p *libnet.Session) Handle(data []byte, session *Session) error {
  if string(data) != "hello" {
    return errors.New("echo failed")
  }
  return nil
}
client, err := libnet.Dial("tcp", "127.0.0.1:8888")
client.Send(libnet.String("hello"))
client.ProcessOnce(client)

client.Close()
```



## Server

Server是对net.Listener的封装，提供更高级别更简易的操作

```golang
Listen(network, address string) (*Server, error)
```

比如：

```golang
server, err := libnet.Listen("tcp", "192.168.1.100:8888")
```

接口很简单，只有如下三个

- Serve

```golang
func (server *Server)Serve(handler IPacketProc) error
```

阻塞accept客户端连接，然后为每个连接创建goroutine进行处理，用户自定义协议处理`IPacketProc`

- Broadcast

```golang
func (server *Server)Broadcast(encoder Encoder) ([]BroadcastWork, error)
```

向该Server的所有客户端广播消息，消息仅编码一次，所以比单独每个客户端发送效率要高，可以根据返回值查询返回状态，确定是否重发

```golang
bw, err := server.Broadcast(libnet.Json(cmd))
if err != nil {
  return
}
for _, v := range {
  err := v.Wait()
  if err == nil {
    // 异步发送成功
  } else if err == SendToClosedError {
    // 异步发送失败，连接关闭，无须重发
  } else if err == AsyncSendTimeoutError {
    // 异步发送失败，超时(5s)，业务决定是否重发
    v.Session.Send(libnet.Json(cmd))
  } else {
    // 其他错误	
  }
}
```



- Stop

```golang
func (server *Server)Stop()
```

停止服务，关闭所有客户端



- Echo服务端例子：

```golang
import "github.com/alvin921/go-libnet"

func (p *libnet.Server) Handle(data []byte, session *Session) error {
  return session.Send(libnet.Bytes(data))
}
server, err := libnet.Listen("tcp", "127.0.0.1:8888")
server.Serve(server)
```



## Channel

Server的Broadcast是向所有客户端连接广播消息，某些场合需要向某些特定客户端广播，所以就有了Channel，对某一类topic感兴趣的session可注册到同一个Channel，发送到该Channel的消息则所有在线session都能收到

创建Channel：

```golang
func NewChannel() *Channel
```

向Channel中加入Session，用于客户端主动注册，kickCallback在Channel.Kick(session)时调用，其适用场景：比如群组管理员踢出某成员，通过kickCallback给被踢出者发送通知

```golang
(channel *Channel) Join(session *Session, kickCallback func())
```

退出Channel，用于session主动退出，或session关闭时调用

```golang
func (channel *Channel) Exit(session *Session)
```

将session踢出Channel

```
func (channel *Channel) Kick(session *Session)
```

Channel成员数目

```golang
func (channel *Channel) Len() int
```

向注册到Channel的所有Session广播消息

```golang
func (channel *Channel) Broadcast(encoder Encoder) ([]BroadcastWork, error)
```



