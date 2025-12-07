package protocol

// Ack is sent by the client to acknowledge received patches.
// It serves multiple purposes:
//  1. Garbage collection of patch history on the server
//  2. Flow control (server knows client's processing capacity)
//  3. Detecting client lag
type Ack struct {
	LastSeq uint64 // Last received sequence number
	Window  uint64 // Receive window size (how many more patches client can accept)
}

// EncodeAck encodes an Ack to bytes.
func EncodeAck(ack *Ack) []byte {
	e := NewEncoder()
	EncodeAckTo(e, ack)
	return e.Bytes()
}

// EncodeAckTo encodes an Ack using the provided encoder.
func EncodeAckTo(e *Encoder, ack *Ack) {
	e.WriteUvarint(ack.LastSeq)
	e.WriteUvarint(ack.Window)
}

// DecodeAck decodes an Ack from bytes.
func DecodeAck(data []byte) (*Ack, error) {
	d := NewDecoder(data)
	return DecodeAckFrom(d)
}

// DecodeAckFrom decodes an Ack from a decoder.
func DecodeAckFrom(d *Decoder) (*Ack, error) {
	lastSeq, err := d.ReadUvarint()
	if err != nil {
		return nil, err
	}

	window, err := d.ReadUvarint()
	if err != nil {
		return nil, err
	}

	return &Ack{
		LastSeq: lastSeq,
		Window:  window,
	}, nil
}

// NewAck creates a new Ack with the given sequence and window.
func NewAck(lastSeq, window uint64) *Ack {
	return &Ack{
		LastSeq: lastSeq,
		Window:  window,
	}
}

// DefaultWindow is the default receive window size.
const DefaultWindow = 100
