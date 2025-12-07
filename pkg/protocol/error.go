package protocol

// ErrorCode identifies the type of error.
type ErrorCode uint16

const (
	ErrUnknown         ErrorCode = 0x0000 // Unknown error
	ErrInvalidFrame    ErrorCode = 0x0001 // Malformed frame
	ErrInvalidEvent    ErrorCode = 0x0002 // Malformed event
	ErrHandlerNotFound ErrorCode = 0x0003 // No handler for HID
	ErrHandlerPanic    ErrorCode = 0x0004 // Handler panicked
	ErrSessionExpired  ErrorCode = 0x0005 // Session no longer valid
	ErrRateLimited     ErrorCode = 0x0006 // Too many requests
	ErrServerError     ErrorCode = 0x0100 // Internal server error
	ErrNotAuthorized   ErrorCode = 0x0101 // Not authorized
	ErrNotFound        ErrorCode = 0x0102 // Resource not found
	ErrValidation      ErrorCode = 0x0103 // Validation failed
)

// String returns the string representation of the error code.
func (ec ErrorCode) String() string {
	switch ec {
	case ErrUnknown:
		return "Unknown"
	case ErrInvalidFrame:
		return "InvalidFrame"
	case ErrInvalidEvent:
		return "InvalidEvent"
	case ErrHandlerNotFound:
		return "HandlerNotFound"
	case ErrHandlerPanic:
		return "HandlerPanic"
	case ErrSessionExpired:
		return "SessionExpired"
	case ErrRateLimited:
		return "RateLimited"
	case ErrServerError:
		return "ServerError"
	case ErrNotAuthorized:
		return "NotAuthorized"
	case ErrNotFound:
		return "NotFound"
	case ErrValidation:
		return "Validation"
	default:
		return "Unknown"
	}
}

// ErrorMessage is sent when an error occurs.
type ErrorMessage struct {
	Code    ErrorCode // Error code
	Message string    // Human-readable error message
	Fatal   bool      // If true, connection should be closed
}

// EncodeErrorMessage encodes an ErrorMessage to bytes.
func EncodeErrorMessage(em *ErrorMessage) []byte {
	e := NewEncoder()
	EncodeErrorMessageTo(e, em)
	return e.Bytes()
}

// EncodeErrorMessageTo encodes an ErrorMessage using the provided encoder.
func EncodeErrorMessageTo(e *Encoder, em *ErrorMessage) {
	e.WriteUint16(uint16(em.Code))
	e.WriteString(em.Message)
	e.WriteBool(em.Fatal)
}

// DecodeErrorMessage decodes an ErrorMessage from bytes.
func DecodeErrorMessage(data []byte) (*ErrorMessage, error) {
	d := NewDecoder(data)
	return DecodeErrorMessageFrom(d)
}

// DecodeErrorMessageFrom decodes an ErrorMessage from a decoder.
func DecodeErrorMessageFrom(d *Decoder) (*ErrorMessage, error) {
	code, err := d.ReadUint16()
	if err != nil {
		return nil, err
	}

	message, err := d.ReadString()
	if err != nil {
		return nil, err
	}

	fatal, err := d.ReadBool()
	if err != nil {
		return nil, err
	}

	return &ErrorMessage{
		Code:    ErrorCode(code),
		Message: message,
		Fatal:   fatal,
	}, nil
}

// NewError creates a new non-fatal ErrorMessage.
func NewError(code ErrorCode, message string) *ErrorMessage {
	return &ErrorMessage{
		Code:    code,
		Message: message,
		Fatal:   false,
	}
}

// NewFatalError creates a new fatal ErrorMessage.
func NewFatalError(code ErrorCode, message string) *ErrorMessage {
	return &ErrorMessage{
		Code:    code,
		Message: message,
		Fatal:   true,
	}
}

// Error implements the error interface.
func (em *ErrorMessage) Error() string {
	if em.Fatal {
		return "fatal: " + em.Code.String() + ": " + em.Message
	}
	return em.Code.String() + ": " + em.Message
}

// IsFatal returns true if this error should close the connection.
func (em *ErrorMessage) IsFatal() bool {
	return em.Fatal
}
