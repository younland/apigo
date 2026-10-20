package main

// kind is the protocol a request speaks.
type kind string

const (
	kindREST      kind = "rest"
	kindGraphQL   kind = "graphql"
	kindWebSocket kind = "websocket"
	kindSSE       kind = "sse"
	kindSocketIO  kind = "socketio"
	kindMQTT      kind = "mqtt"
)

// requestKinds are the kinds a collection's request can be, in the order
// the new-request menu offers them.
var requestKinds = []kind{kindREST, kindGraphQL, kindWebSocket, kindSSE, kindSocketIO, kindMQTT}

// kindName is the name a kind shows.
func kindName(k kind) string {
	switch k {
	case kindGraphQL:
		return "GraphQL"
	case kindWebSocket:
		return "WebSocket"
	case kindSSE:
		return "SSE"
	case kindSocketIO:
		return "Socket.IO"
	case kindMQTT:
		return "MQTT"
	default:
		return "REST"
	}
}
