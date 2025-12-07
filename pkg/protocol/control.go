package protocol

// ControlType identifies the type of control message.
type ControlType uint8

const (
	ControlPing          ControlType = 0x01 // Client/server ping
	ControlPong          ControlType = 0x02 // Response to ping
	ControlResyncRequest ControlType = 0x10 // Client requests missed patches
	ControlResyncPatches ControlType = 0x11 // Server sends missed patches
	ControlResyncFull    ControlType = 0x12 // Server sends full HTML reload
	ControlClose         ControlType = 0x20 // Session close
)

// String returns the string representation of the control type.
func (ct ControlType) String() string {
	switch ct {
	case ControlPing:
		return "Ping"
	case ControlPong:
		return "Pong"
	case ControlResyncRequest:
		return "ResyncRequest"
	case ControlResyncPatches:
		return "ResyncPatches"
	case ControlResyncFull:
		return "ResyncFull"
	case ControlClose:
		return "Close"
	default:
		return "Unknown"
	}
}

// CloseReason indicates why a session is being closed.
type CloseReason uint8

const (
	CloseNormal         CloseReason = 0x00 // Normal closure
	CloseGoingAway      CloseReason = 0x01 // Client/server going away
	CloseSessionExpired CloseReason = 0x02 // Session expired
	CloseServerShutdown CloseReason = 0x03 // Server shutting down
	CloseError          CloseReason = 0x04 // Error occurred
)

// String returns the string representation of the close reason.
func (cr CloseReason) String() string {
	switch cr {
	case CloseNormal:
		return "Normal"
	case CloseGoingAway:
		return "GoingAway"
	case CloseSessionExpired:
		return "SessionExpired"
	case CloseServerShutdown:
		return "ServerShutdown"
	case CloseError:
		return "Error"
	default:
		return "Unknown"
	}
}

// PingPong is the payload for Ping and Pong messages.
type PingPong struct {
	Timestamp uint64 // Unix timestamp in milliseconds
}

// ResyncRequest is sent by client to request missed patches.
type ResyncRequest struct {
	LastSeq uint64 // Last received sequence number
}

// ResyncResponse is sent by server with missed patches or full reload.
type ResyncResponse struct {
	Type    ControlType // ResyncPatches or ResyncFull
	FromSeq uint64      // Starting sequence number (for ResyncPatches)
	Patches []Patch     // Missed patches (for ResyncPatches)
	HTML    string      // Full HTML (for ResyncFull)
}

// CloseMessage is sent when closing a session.
type CloseMessage struct {
	Reason  CloseReason
	Message string
}

// EncodeControl encodes a control message to bytes.
func EncodeControl(ct ControlType, payload any) []byte {
	e := NewEncoder()
	EncodeControlTo(e, ct, payload)
	return e.Bytes()
}

// EncodeControlTo encodes a control message using the provided encoder.
func EncodeControlTo(e *Encoder, ct ControlType, payload any) {
	e.WriteByte(byte(ct))

	switch ct {
	case ControlPing, ControlPong:
		if pp, ok := payload.(*PingPong); ok {
			e.WriteUint64(pp.Timestamp)
		} else {
			e.WriteUint64(0)
		}

	case ControlResyncRequest:
		if rr, ok := payload.(*ResyncRequest); ok {
			e.WriteUvarint(rr.LastSeq)
		} else {
			e.WriteUvarint(0)
		}

	case ControlResyncPatches:
		if rr, ok := payload.(*ResyncResponse); ok {
			e.WriteUvarint(rr.FromSeq)
			e.WriteUvarint(uint64(len(rr.Patches)))
			for i := range rr.Patches {
				encodePatch(e, &rr.Patches[i])
			}
		} else {
			e.WriteUvarint(0)
			e.WriteUvarint(0)
		}

	case ControlResyncFull:
		if rr, ok := payload.(*ResyncResponse); ok {
			e.WriteString(rr.HTML)
		} else {
			e.WriteString("")
		}

	case ControlClose:
		if cm, ok := payload.(*CloseMessage); ok {
			e.WriteByte(byte(cm.Reason))
			e.WriteString(cm.Message)
		} else {
			e.WriteByte(byte(CloseNormal))
			e.WriteString("")
		}
	}
}

// DecodeControl decodes a control message from bytes.
// Returns the control type and the decoded payload.
func DecodeControl(data []byte) (ControlType, any, error) {
	d := NewDecoder(data)
	return DecodeControlFrom(d)
}

// DecodeControlFrom decodes a control message from a decoder.
func DecodeControlFrom(d *Decoder) (ControlType, any, error) {
	typeByte, err := d.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	ct := ControlType(typeByte)

	switch ct {
	case ControlPing, ControlPong:
		ts, err := d.ReadUint64()
		if err != nil {
			return ct, nil, err
		}
		return ct, &PingPong{Timestamp: ts}, nil

	case ControlResyncRequest:
		lastSeq, err := d.ReadUvarint()
		if err != nil {
			return ct, nil, err
		}
		return ct, &ResyncRequest{LastSeq: lastSeq}, nil

	case ControlResyncPatches:
		fromSeq, err := d.ReadUvarint()
		if err != nil {
			return ct, nil, err
		}
		count, err := d.ReadUvarint()
		if err != nil {
			return ct, nil, err
		}
		patches := make([]Patch, count)
		for i := uint64(0); i < count; i++ {
			if err := decodePatch(d, &patches[i]); err != nil {
				return ct, nil, err
			}
		}
		return ct, &ResyncResponse{
			Type:    ControlResyncPatches,
			FromSeq: fromSeq,
			Patches: patches,
		}, nil

	case ControlResyncFull:
		html, err := d.ReadString()
		if err != nil {
			return ct, nil, err
		}
		return ct, &ResyncResponse{
			Type: ControlResyncFull,
			HTML: html,
		}, nil

	case ControlClose:
		reason, err := d.ReadByte()
		if err != nil {
			return ct, nil, err
		}
		message, err := d.ReadString()
		if err != nil {
			return ct, nil, err
		}
		return ct, &CloseMessage{
			Reason:  CloseReason(reason),
			Message: message,
		}, nil

	default:
		return ct, nil, nil
	}
}

// NewPing creates a new Ping message.
func NewPing(timestamp uint64) (ControlType, *PingPong) {
	return ControlPing, &PingPong{Timestamp: timestamp}
}

// NewPong creates a new Pong message.
func NewPong(timestamp uint64) (ControlType, *PingPong) {
	return ControlPong, &PingPong{Timestamp: timestamp}
}

// NewResyncRequest creates a new ResyncRequest message.
func NewResyncRequest(lastSeq uint64) (ControlType, *ResyncRequest) {
	return ControlResyncRequest, &ResyncRequest{LastSeq: lastSeq}
}

// NewResyncPatches creates a new ResyncPatches response.
func NewResyncPatches(fromSeq uint64, patches []Patch) (ControlType, *ResyncResponse) {
	return ControlResyncPatches, &ResyncResponse{
		Type:    ControlResyncPatches,
		FromSeq: fromSeq,
		Patches: patches,
	}
}

// NewResyncFull creates a new ResyncFull response.
func NewResyncFull(html string) (ControlType, *ResyncResponse) {
	return ControlResyncFull, &ResyncResponse{
		Type: ControlResyncFull,
		HTML: html,
	}
}

// NewClose creates a new Close message.
func NewClose(reason CloseReason, message string) (ControlType, *CloseMessage) {
	return ControlClose, &CloseMessage{Reason: reason, Message: message}
}
