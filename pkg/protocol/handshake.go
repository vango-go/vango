package protocol

// HandshakeStatus represents the result of a handshake.
type HandshakeStatus uint8

const (
	HandshakeOK              HandshakeStatus = 0x00
	HandshakeVersionMismatch HandshakeStatus = 0x01
	HandshakeInvalidCSRF     HandshakeStatus = 0x02
	HandshakeSessionExpired  HandshakeStatus = 0x03
	HandshakeServerBusy      HandshakeStatus = 0x04
	HandshakeUpgradeRequired HandshakeStatus = 0x05
	HandshakeInvalidFormat   HandshakeStatus = 0x06 // Malformed handshake message
	HandshakeNotAuthorized   HandshakeStatus = 0x07 // Authentication failed
	HandshakeInternalError   HandshakeStatus = 0x08 // Server error
)

// String returns the string representation of the handshake status.
func (hs HandshakeStatus) String() string {
	switch hs {
	case HandshakeOK:
		return "OK"
	case HandshakeVersionMismatch:
		return "VersionMismatch"
	case HandshakeInvalidCSRF:
		return "InvalidCSRF"
	case HandshakeSessionExpired:
		return "SessionExpired"
	case HandshakeServerBusy:
		return "ServerBusy"
	case HandshakeUpgradeRequired:
		return "UpgradeRequired"
	case HandshakeInvalidFormat:
		return "InvalidFormat"
	case HandshakeNotAuthorized:
		return "NotAuthorized"
	case HandshakeInternalError:
		return "InternalError"
	default:
		return "Unknown"
	}
}

// ProtocolVersion represents a protocol version as major.minor.
type ProtocolVersion struct {
	Major uint8
	Minor uint8
}

// CurrentVersion is the current protocol version.
var CurrentVersion = ProtocolVersion{Major: 2, Minor: 0}

// ClientHello is sent by the client after WebSocket connection is established.
type ClientHello struct {
	Version   ProtocolVersion // Protocol version
	CSRFToken string          // CSRF token for validation
	SessionID string          // Existing session ID (empty if new)
	LastSeq   uint32          // Last seen sequence number
	ViewportW uint16          // Viewport width
	ViewportH uint16          // Viewport height
	TZOffset  int16           // Timezone offset in minutes from UTC
}

// ServerHello is the server's response to ClientHello.
type ServerHello struct {
	Status     HandshakeStatus // Handshake result
	SessionID  string          // Session ID (new or existing)
	NextSeq    uint32          // Next expected sequence number
	ServerTime uint64          // Server time in Unix milliseconds
	Flags      uint16          // Server capability flags
}

// Server capability flags.
const (
	ServerFlagCompression uint16 = 0x0001 // Server supports compression
	ServerFlagBinaryBlobs uint16 = 0x0002 // Server supports binary blob uploads
	ServerFlagStreaming   uint16 = 0x0004 // Server supports streaming responses
)

// EncodeClientHello encodes a ClientHello to bytes.
func EncodeClientHello(ch *ClientHello) []byte {
	e := NewEncoder()
	EncodeClientHelloTo(e, ch)
	return e.Bytes()
}

// EncodeClientHelloTo encodes a ClientHello using the provided encoder.
func EncodeClientHelloTo(e *Encoder, ch *ClientHello) {
	e.WriteByte(ch.Version.Major)
	e.WriteByte(ch.Version.Minor)
	e.WriteString(ch.CSRFToken)
	e.WriteString(ch.SessionID)
	e.WriteUint32(ch.LastSeq)
	e.WriteUint16(ch.ViewportW)
	e.WriteUint16(ch.ViewportH)
	e.WriteInt16(ch.TZOffset)
}

// DecodeClientHello decodes a ClientHello from bytes.
func DecodeClientHello(data []byte) (*ClientHello, error) {
	d := NewDecoder(data)
	return DecodeClientHelloFrom(d)
}

// DecodeClientHelloFrom decodes a ClientHello from a decoder.
func DecodeClientHelloFrom(d *Decoder) (*ClientHello, error) {
	ch := &ClientHello{}
	var err error

	major, err := d.ReadByte()
	if err != nil {
		return nil, err
	}
	minor, err := d.ReadByte()
	if err != nil {
		return nil, err
	}
	ch.Version = ProtocolVersion{Major: major, Minor: minor}

	ch.CSRFToken, err = d.ReadString()
	if err != nil {
		return nil, err
	}

	ch.SessionID, err = d.ReadString()
	if err != nil {
		return nil, err
	}

	ch.LastSeq, err = d.ReadUint32()
	if err != nil {
		return nil, err
	}

	ch.ViewportW, err = d.ReadUint16()
	if err != nil {
		return nil, err
	}

	ch.ViewportH, err = d.ReadUint16()
	if err != nil {
		return nil, err
	}

	ch.TZOffset, err = d.ReadInt16()
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// EncodeServerHello encodes a ServerHello to bytes.
func EncodeServerHello(sh *ServerHello) []byte {
	e := NewEncoder()
	EncodeServerHelloTo(e, sh)
	return e.Bytes()
}

// EncodeServerHelloTo encodes a ServerHello using the provided encoder.
func EncodeServerHelloTo(e *Encoder, sh *ServerHello) {
	e.WriteByte(byte(sh.Status))
	e.WriteString(sh.SessionID)
	e.WriteUint32(sh.NextSeq)
	e.WriteUint64(sh.ServerTime)
	e.WriteUint16(sh.Flags)
}

// DecodeServerHello decodes a ServerHello from bytes.
func DecodeServerHello(data []byte) (*ServerHello, error) {
	d := NewDecoder(data)
	return DecodeServerHelloFrom(d)
}

// DecodeServerHelloFrom decodes a ServerHello from a decoder.
func DecodeServerHelloFrom(d *Decoder) (*ServerHello, error) {
	sh := &ServerHello{}
	var err error

	status, err := d.ReadByte()
	if err != nil {
		return nil, err
	}
	sh.Status = HandshakeStatus(status)

	sh.SessionID, err = d.ReadString()
	if err != nil {
		return nil, err
	}

	sh.NextSeq, err = d.ReadUint32()
	if err != nil {
		return nil, err
	}

	sh.ServerTime, err = d.ReadUint64()
	if err != nil {
		return nil, err
	}

	sh.Flags, err = d.ReadUint16()
	if err != nil {
		return nil, err
	}

	return sh, nil
}

// NewClientHello creates a new ClientHello with default version.
func NewClientHello(csrfToken string) *ClientHello {
	return &ClientHello{
		Version:   CurrentVersion,
		CSRFToken: csrfToken,
	}
}

// NewServerHello creates a new successful ServerHello.
func NewServerHello(sessionID string, nextSeq uint32, serverTime uint64) *ServerHello {
	return &ServerHello{
		Status:     HandshakeOK,
		SessionID:  sessionID,
		NextSeq:    nextSeq,
		ServerTime: serverTime,
	}
}

// NewServerHelloError creates a ServerHello with an error status.
func NewServerHelloError(status HandshakeStatus) *ServerHello {
	return &ServerHello{
		Status: status,
	}
}
