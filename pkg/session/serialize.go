package session

import (
	"encoding/json"
	"time"
)

// SerializableSession is the JSON-serializable representation of a session.
// This structure is used for persistence across server restarts.
type SerializableSession struct {
	// ID is the unique session identifier.
	ID string `json:"id"`

	// UserID is the authenticated user ID, if any.
	UserID string `json:"user_id,omitempty"`

	// CreatedAt is when the session was created.
	CreatedAt time.Time `json:"created_at"`

	// LastActive is when the session was last active.
	LastActive time.Time `json:"last_active"`

	// Values contains Session.Get/Set values.
	Values map[string]json.RawMessage `json:"values,omitempty"`

	// Signals contains persisted signal values by key.
	// Transient signals are excluded.
	Signals map[string]json.RawMessage `json:"signals,omitempty"`

	// Route is the current page route.
	Route string `json:"route,omitempty"`

	// RouteParams contains the current route parameters.
	RouteParams map[string]string `json:"route_params,omitempty"`

	// Version is the serialization format version.
	Version int `json:"version"`
}

// CurrentSerializationVersion is the current version of the serialization format.
// Increment when making breaking changes to the format.
const CurrentSerializationVersion = 1

// Serialize converts a SerializableSession to bytes.
func Serialize(ss *SerializableSession) ([]byte, error) {
	ss.Version = CurrentSerializationVersion
	return json.Marshal(ss)
}

// Deserialize converts bytes back to a SerializableSession.
func Deserialize(data []byte) (*SerializableSession, error) {
	var ss SerializableSession
	if err := json.Unmarshal(data, &ss); err != nil {
		return nil, err
	}
	return &ss, nil
}

// SignalConfig holds configuration for signal persistence.
// This is used by the vango.Signal to track persistence options.
type SignalConfig struct {
	// Transient signals are not persisted to the store.
	Transient bool

	// PersistKey is the explicit key for serialization.
	// If empty, an auto-generated key is used based on component/position.
	PersistKey string
}

// NewSignalConfig creates a default SignalConfig.
func NewSignalConfig() *SignalConfig {
	return &SignalConfig{}
}
