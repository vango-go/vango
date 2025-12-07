package protocol

import (
	"math"
	"testing"
)

func TestEncodeDecodeUvarint(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		bytes int // expected encoded length
	}{
		{"zero", 0, 1},
		{"one", 1, 1},
		{"max_1byte", 127, 1},
		{"min_2byte", 128, 2},
		{"max_2byte", 16383, 2},
		{"min_3byte", 16384, 3},
		{"medium", 1000000, 3},
		{"large", 1<<28, 5},
		{"max_uint32", math.MaxUint32, 5},
		{"max_uint64", math.MaxUint64, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, MaxVarintLen)
			n := EncodeUvarint(buf, tc.value)

			if n != tc.bytes {
				t.Errorf("EncodeUvarint(%d) = %d bytes, want %d", tc.value, n, tc.bytes)
			}

			decoded, read := DecodeUvarint(buf[:n])
			if read != n {
				t.Errorf("DecodeUvarint read %d bytes, want %d", read, n)
			}
			if decoded != tc.value {
				t.Errorf("DecodeUvarint = %d, want %d", decoded, tc.value)
			}
		})
	}
}

func TestEncodeDecodeSvarint(t *testing.T) {
	tests := []struct {
		name  string
		value int64
	}{
		{"zero", 0},
		{"one", 1},
		{"neg_one", -1},
		{"small_pos", 100},
		{"small_neg", -100},
		{"medium_pos", 1000000},
		{"medium_neg", -1000000},
		{"max_int32", math.MaxInt32},
		{"min_int32", math.MinInt32},
		{"max_int64", math.MaxInt64},
		{"min_int64", math.MinInt64},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, MaxVarintLen)
			n := EncodeSvarint(buf, tc.value)

			decoded, read := DecodeSvarint(buf[:n])
			if read != n {
				t.Errorf("DecodeSvarint read %d bytes, want %d", read, n)
			}
			if decoded != tc.value {
				t.Errorf("DecodeSvarint = %d, want %d", decoded, tc.value)
			}
		})
	}
}

func TestUvarintLen(t *testing.T) {
	tests := []struct {
		value    uint64
		expected int
	}{
		{0, 1},
		{127, 1},
		{128, 2},
		{16383, 2},
		{16384, 3},
		{math.MaxUint32, 5},
		{math.MaxUint64, 10},
	}

	for _, tc := range tests {
		got := UvarintLen(tc.value)
		if got != tc.expected {
			t.Errorf("UvarintLen(%d) = %d, want %d", tc.value, got, tc.expected)
		}

		// Verify against actual encoding
		buf := make([]byte, MaxVarintLen)
		actual := EncodeUvarint(buf, tc.value)
		if got != actual {
			t.Errorf("UvarintLen(%d) = %d, but EncodeUvarint wrote %d bytes", tc.value, got, actual)
		}
	}
}

func TestSvarintLen(t *testing.T) {
	tests := []struct {
		value    int64
		expected int
	}{
		{0, 1},
		{-1, 1},
		{1, 1},
		{-64, 1},
		{63, 1},
		{-65, 2},
		{64, 2},
		{math.MaxInt64, 10},
		{math.MinInt64, 10},
	}

	for _, tc := range tests {
		got := SvarintLen(tc.value)
		if got != tc.expected {
			t.Errorf("SvarintLen(%d) = %d, want %d", tc.value, got, tc.expected)
		}

		// Verify against actual encoding
		buf := make([]byte, MaxVarintLen)
		actual := EncodeSvarint(buf, tc.value)
		if got != actual {
			t.Errorf("SvarintLen(%d) = %d, but EncodeSvarint wrote %d bytes", tc.value, got, actual)
		}
	}
}

func TestDecodeUvarintErrors(t *testing.T) {
	// Empty buffer
	_, n := DecodeUvarint([]byte{})
	if n >= 0 {
		t.Error("DecodeUvarint(empty) should return negative")
	}

	// Incomplete varint (all continuation bits set)
	_, n = DecodeUvarint([]byte{0x80, 0x80, 0x80})
	if n >= 0 {
		t.Error("DecodeUvarint(incomplete) should return negative")
	}

	// Overflow (11 continuation bytes)
	overflow := make([]byte, 11)
	for i := range overflow {
		overflow[i] = 0x80
	}
	_, n = DecodeUvarint(overflow)
	if n != -2 {
		t.Errorf("DecodeUvarint(overflow) = %d, want -2", n)
	}
}

func TestZigZagEncoding(t *testing.T) {
	// Test that ZigZag encoding produces expected values
	// 0 -> 0, -1 -> 1, 1 -> 2, -2 -> 3, 2 -> 4, etc.
	tests := []struct {
		signed   int64
		unsigned uint64
	}{
		{0, 0},
		{-1, 1},
		{1, 2},
		{-2, 3},
		{2, 4},
		{-3, 5},
		{3, 6},
	}

	for _, tc := range tests {
		buf := make([]byte, MaxVarintLen)
		EncodeSvarint(buf, tc.signed)

		// Decode as unsigned to verify ZigZag mapping
		decoded, _ := DecodeUvarint(buf)
		if decoded != tc.unsigned {
			t.Errorf("ZigZag(%d) = %d, want %d", tc.signed, decoded, tc.unsigned)
		}
	}
}

func BenchmarkEncodeUvarint(b *testing.B) {
	buf := make([]byte, MaxVarintLen)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncodeUvarint(buf, uint64(i))
	}
}

func BenchmarkDecodeUvarint(b *testing.B) {
	buf := make([]byte, MaxVarintLen)
	EncodeUvarint(buf, 12345678)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeUvarint(buf)
	}
}

func BenchmarkEncodeSvarint(b *testing.B) {
	buf := make([]byte, MaxVarintLen)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncodeSvarint(buf, int64(i)-int64(b.N/2))
	}
}

func BenchmarkDecodeSvarint(b *testing.B) {
	buf := make([]byte, MaxVarintLen)
	EncodeSvarint(buf, -12345678)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeSvarint(buf)
	}
}
