// Package session provides session persistence and management for Vango.
//
// This package implements pluggable session storage backends, serialization,
// and memory protection mechanisms for server-driven web applications.
//
// # Session Storage
//
// The SessionStore interface defines the contract for session persistence:
//
//	store := session.NewRedisStore(redisClient)
//	// or
//	store := session.NewSQLStore(db)
//	// or (default)
//	store := session.NewMemoryStore()
//
// # Session Serialization
//
// Sessions can be serialized to bytes for persistence:
//
//	data, err := sess.Serialize()
//	// Later...
//	err := sess.Deserialize(data)
//
// # Memory Protection
//
// The Manager provides LRU eviction and per-IP limits:
//
//	manager := session.NewManager(session.ManagerConfig{
//	    MaxDetached:      10000,
//	    MaxPerIP:         100,
//	    Store:            store,
//	    EvictionPolicy:   session.EvictionLRU,
//	})
//
// # Signal Persistence Options
//
// Signals can be configured for persistence behavior:
//
//	cursor := vango.Signal(Point{0, 0}, session.Transient())     // Not persisted
//	userID := vango.Signal(0, session.PersistKey("user_id"))    // Persisted with key
//	formData := vango.Signal(Form{})                             // Persisted with auto-key
package session
