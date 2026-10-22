// Package ws speaks WebSocket to a server: it dials one, keeps what comes back
// as events a log shows, and sends messages as text or bytes.
package ws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EventKind is what an event says happened.
type EventKind int

const (
	// Connected is a socket that opened.
	Connected EventKind = iota
	// Disconnected is a socket that closed, by the server or by the app.
	Disconnected
	// Failed is a connection that could not be made, or a socket that ended
	// with an error.
	Failed
	// Sent is a message the app sent.
	Sent
	// Received is a message that came back.
	Received
)

// Event is one line of the log: a connection, a closing, a message, or a
// failure.
type Event struct {
	Kind EventKind
	At   time.Time
	// Text is the message, or what happened when it carries no payload.
	Text string
	// Binary says the message was bytes rather than text.
	Binary bool
	// Size is how many bytes the message took.
	Size int
}

// Conn is an open socket. Its methods are safe to call from any goroutine.
type Conn struct {
	socket *websocket.Conn
	events chan Event
	// write serialises sends: gorilla allows one writer at a time.
	write sync.Mutex
	once  sync.Once
}

// Dial opens a socket at the address, sending the headers and asking for the
// subprotocols the request names. The socket reports what it receives and how
// it ends through Events.
func Dial(ctx context.Context, address string, header http.Header, protocols []string) (*Conn, error) {
	dialer := &websocket.Dialer{
		Subprotocols:     protocols,
		HandshakeTimeout: 10 * time.Second,
	}
	socket, resp, err := dialer.DialContext(ctx, address, header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("%s（HTTP %d）", connectError(err), resp.StatusCode)
		}
		return nil, connectError(err)
	}
	c := &Conn{socket: socket, events: make(chan Event, 64)}
	go c.read()
	return c, nil
}

// connectError says why a socket did not open, in words the log reads well.
func connectError(err error) error {
	var handshake websocket.HandshakeError
	if errors.As(err, &handshake) {
		return errors.New("握手失败：对方不是 WebSocket 服务")
	}
	return fmt.Errorf("连接失败：%w", err)
}

// read takes what the server sends until the socket ends, and reports it.
func (c *Conn) read() {
	defer close(c.events)
	for {
		kind, data, err := c.socket.ReadMessage()
		if err != nil {
			c.events <- Event{Kind: Disconnected, At: time.Now(), Text: closeText(err)}
			return
		}
		c.events <- Event{
			Kind:   Received,
			At:     time.Now(),
			Text:   string(data),
			Binary: kind == websocket.BinaryMessage,
			Size:   len(data),
		}
	}
}

// closeText says why a socket ended, as the log tells it.
func closeText(err error) string {
	var closed *websocket.CloseError
	if errors.As(err, &closed) {
		if closed.Text != "" {
			return fmt.Sprintf("已断开（%d %s）", closed.Code, closed.Text)
		}
		return fmt.Sprintf("已断开（%d）", closed.Code)
	}
	return "已断开：" + err.Error()
}

// Send writes a message: text as it stands, or bytes for a binary one.
func (c *Conn) Send(ctx context.Context, binary bool, data []byte) error {
	c.write.Lock()
	defer c.write.Unlock()
	kind := websocket.TextMessage
	if binary {
		kind = websocket.BinaryMessage
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.socket.SetWriteDeadline(deadline)
	} else {
		_ = c.socket.SetWriteDeadline(time.Now().Add(10 * time.Second))
	}
	if err := c.socket.WriteMessage(kind, data); err != nil {
		return fmt.Errorf("发送失败：%w", err)
	}
	return nil
}

// Events is what the socket received and how it ends: a received message, or a
// closing with its reason, after which the channel is closed.
func (c *Conn) Events() <-chan Event { return c.events }

// Subprotocol is the subprotocol the server agreed to, "" when it agreed to
// none.
func (c *Conn) Subprotocol() string { return c.socket.Subprotocol() }

// Close shuts the socket down, which ends the reading goroutine too.
func (c *Conn) Close() error {
	var err error
	c.once.Do(func() {
		// A closing handshake the server never answers must not hang the app.
		_ = c.socket.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second))
		err = c.socket.Close()
	})
	return err
}
