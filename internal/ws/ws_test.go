package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// echoServer is a WebSocket server that sends back what it reads, and answers
// the handshake with the subprotocol the test asks for.
type echoServer struct {
	server *httptest.Server
	seen   chan requestSeen
}

// requestSeen is what the server read of the handshake.
type requestSeen struct {
	header      http.Header
	subprotocol string
}

func newEchoServer(t *testing.T) *echoServer {
	t.Helper()
	e := &echoServer{seen: make(chan requestSeen, 4)}
	upgrader := websocket.Upgrader{
		Subprotocols: []string{"chat"},
		CheckOrigin:  func(*http.Request) bool { return true },
	}
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer socket.Close()
		e.seen <- requestSeen{header: r.Header.Clone(), subprotocol: socket.Subprotocol()}
		for {
			kind, data, err := socket.ReadMessage()
			if err != nil {
				return
			}
			if err := socket.WriteMessage(kind, data); err != nil {
				return
			}
		}
	}))
	t.Cleanup(e.server.Close)
	return e
}

// address is the socket address of the server.
func (e *echoServer) address() string {
	return "ws" + strings.TrimPrefix(e.server.URL, "http")
}

// next waits for the next event of a kind, or fails the test.
func next(t *testing.T, events <-chan Event, want EventKind) Event {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatalf("the socket closed while waiting for kind %v", want)
			}
			if event.Kind == want {
				return event
			}
		case <-deadline:
			t.Fatalf("no event of kind %v within five seconds", want)
		}
	}
}

// TestEchoText checks a socket sends a message and reports what comes back.
func TestEchoText(t *testing.T) {
	server := newEchoServer(t)
	conn, err := Dial(context.Background(), server.address(), http.Header{"X-Token": {"t"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	seen := <-server.seen
	if seen.header.Get("X-Token") != "t" {
		t.Errorf("handshake headers %v, want X-Token", seen.header)
	}
	if err := conn.Send(context.Background(), false, []byte("你好")); err != nil {
		t.Fatal(err)
	}
	event := next(t, conn.Events(), Received)
	if event.Text != "你好" || event.Binary || event.Size != len("你好") {
		t.Errorf("event %+v, want the text echoed back", event)
	}
}

// TestEchoBinary checks bytes come back as bytes.
func TestEchoBinary(t *testing.T) {
	server := newEchoServer(t)
	conn, err := Dial(context.Background(), server.address(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.Send(context.Background(), true, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	event := next(t, conn.Events(), Received)
	if !event.Binary || event.Size != 3 {
		t.Errorf("event %+v, want three bytes back", event)
	}
}

// TestSubprotocol checks the handshake asks for the subprotocol the request
// names, and reports the one agreed.
func TestSubprotocol(t *testing.T) {
	server := newEchoServer(t)
	conn, err := Dial(context.Background(), server.address(), nil, []string{"chat", "other"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	seen := <-server.seen
	if seen.subprotocol != "chat" || conn.Subprotocol() != "chat" {
		t.Errorf("subprotocol %q / %q, want chat", seen.subprotocol, conn.Subprotocol())
	}
}

// TestCloseIsReported checks a closed socket says so, once.
func TestCloseIsReported(t *testing.T) {
	server := newEchoServer(t)
	conn, err := Dial(context.Background(), server.address(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	next(t, conn.Events(), Disconnected)
	if _, open := <-conn.Events(); open {
		t.Error("the events of a closed socket are still open, want the channel closed")
	}
}

// TestDialReportsAFailedHandshake checks a server that is no WebSocket service
// is told as such.
func TestDialReportsAFailedHandshake(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	address := "ws" + strings.TrimPrefix(server.URL, "http")
	_, err := Dial(context.Background(), address, nil, nil)
	if err == nil {
		t.Fatal("no error for a handshake the server refused")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error %v, want the status the server answered", err)
	}
}
