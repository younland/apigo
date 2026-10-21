package app

import "apigo/internal/store"

// The collection model lives in internal/store. These aliases keep the views
// and the tree code reading by the names the app has always used.
type (
	collection   = store.Collection
	folder       = store.Folder
	savedRequest = store.Request
	savedRow     = store.Row
	environment  = store.Environment
	historyEntry = store.HistoryEntry
	settings     = store.Settings
	kind         = store.Kind
	dataStore    = store.Store
)

// The kinds of interface are the store's, so what a tree file says and what a
// tab edits are the same type.
const (
	kindREST      = store.KindREST
	kindGraphQL   = store.KindGraphQL
	kindWebSocket = store.KindWebSocket
	kindSSE       = store.KindSSE
	kindSocketIO  = store.KindSocketIO
	kindMQTT      = store.KindMQTT
)
