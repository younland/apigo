package store

// kind is the protocol a request speaks.
type Kind string

const (
	KindREST      Kind = "rest"
	KindGraphQL   Kind = "graphql"
	KindWebSocket Kind = "websocket"
	KindSSE       Kind = "sse"
	KindSocketIO  Kind = "socketio"
	KindMQTT      Kind = "mqtt"
)

// requestKinds are the kinds a collection's request can be, in the order
// the new-request menu offers them.
var RequestKinds = []Kind{KindREST, KindGraphQL, KindWebSocket, KindSSE, KindSocketIO, KindMQTT}

// kindName is the name a kind shows.
func KindName(k Kind) string {
	switch k {
	case KindGraphQL:
		return "GraphQL"
	case KindWebSocket:
		return "WebSocket"
	case KindSSE:
		return "SSE"
	case KindSocketIO:
		return "Socket.IO"
	case KindMQTT:
		return "MQTT"
	default:
		return "REST"
	}
}
