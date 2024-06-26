// Copyright (c) HashiCorp, Inc
// SPDX-License-Identifier: MPL-2.0

package segment

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/polarsignals/wal/types"
	"github.com/stretchr/testify/require"
)

func TestFileHeaderCodec(t *testing.T) {
	cases := []struct {
		name            string
		info            types.SegmentInfo
		bufSize         int
		corrupt         func([]byte) []byte
		wantWriteErr    string
		wantReadErr     string
		wantValidateErr string
	}{
		{
			name: "basic encoding/decoding",
			info: types.SegmentInfo{
				BaseIndex: 1234,
				ID:        4321,
			},
		},
		{
			name: "short buf writing",
			info: types.SegmentInfo{
				BaseIndex: 1234,
				ID:        4321,
			},
			bufSize:      10,
			wantWriteErr: "short buffer",
		},
		{
			name: "short buf reading",
			info: types.SegmentInfo{
				BaseIndex: 1234,
				ID:        4321,
			},
			corrupt: func(buf []byte) []byte {
				return buf[0:5]
			},
			wantReadErr: "short buffer",
		},
		{
			name: "bad magic reading",
			info: types.SegmentInfo{
				BaseIndex: 1234,
				ID:        4321,
			},
			corrupt: func(buf []byte) []byte {
				buf[0] = 0xff
				return buf
			},
			wantReadErr: "corrupt",
		},
		{
			name: "bad BaseIndex reading",
			info: types.SegmentInfo{
				BaseIndex: 1234,
				ID:        4321,
			},
			corrupt: func(buf []byte) []byte {
				buf[8] = 0xff
				return buf
			},
			wantValidateErr: "corrupt",
		},
		{
			name: "bad ID reading",
			info: types.SegmentInfo{
				BaseIndex: 1234,
				ID:        4321,
			},
			corrupt: func(buf []byte) []byte {
				buf[16] = 0xff
				return buf
			},
			wantValidateErr: "corrupt",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			length := fileHeaderLen
			if tc.bufSize > 0 {
				length = tc.bufSize
			}
			buf := make([]byte, length)

			err := writeFileHeader(buf, tc.info)

			if tc.wantWriteErr != "" {
				require.ErrorContains(t, err, tc.wantWriteErr)
				return
			}
			require.NoError(t, err)

			if tc.corrupt != nil {
				buf = tc.corrupt(buf)
			}

			got, err := readFileHeader(buf)
			if tc.wantReadErr != "" {
				require.ErrorContains(t, err, tc.wantReadErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)

			err = validateFileHeader(*got, tc.info)
			if tc.wantValidateErr != "" {
				require.ErrorContains(t, err, tc.wantValidateErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestFileHeaderCodecFuzz(t *testing.T) {
	fzz := fuzz.New()

	var info types.SegmentInfo
	var buf [fileHeaderLen]byte
	for i := 0; i < 1000; i++ {
		fzz.Fuzz(&info)
		err := writeFileHeader(buf[:], info)
		require.NoError(t, err)

		t.Logf("% x", buf[:])

		got, err := readFileHeader(buf[:])
		require.NoError(t, err)
		require.NotNil(t, got)

		err = validateFileHeader(*got, info)
		require.NoError(t, err)
	}
}

func TestFrameCodecFuzz(t *testing.T) {
	fzz := fuzz.New()

	var length uint16
	// Allocate an extra frameHeaderLen heres because some lengths might end up
	// needing padding which takes them just over the buffer size.
	var buf [math.MaxUint16 + frameHeaderLen + frameHeaderLen]byte
	var val = []byte(strings.Repeat("A Value!", math.MaxUint16/8))
	var fh frameHeader
	for i := 0; i < 1000; i++ {
		fzz.Fuzz(&length)

		fh.typ = FrameEntry
		fh.len = uint32(length)

		expectLen := encodedFrameSize(int(length))

		// Note length of val is not the same as fh.len which is what should be
		// used.
		err := writeFrame(buf[:expectLen], fh, val)
		require.NoError(t, err)

		// We mostly care about the start and end...
		if expectLen > 64 {
			t.Logf("% x [...] % x (%d)", buf[0:16], buf[expectLen-16:expectLen], expectLen)
		} else {
			t.Logf("% x", buf[:expectLen])
		}

		// Verify the last padLen bytes are zero
		for i := padLen(int(length)); i > 0; i-- {
			require.Equal(t, byte(0), buf[expectLen-i],
				"expected last %d bytes to be padding. Byte %d of %d isn't zero.",
				padLen(int(length)), expectLen-i, expectLen)
		}

		got, err := readFrameHeader(buf[:])
		require.NoError(t, err)
		require.Equal(t, fh, got)
	}
}

func TestPadLen(t *testing.T) {
	fzz := fuzz.New()
	var length uint32

	for i := 0; i < 1000; i++ {
		fzz.Fuzz(&length)

		got := padLen(int(length))

		t.Log("len", length)

		// Test basic properties of padLen
		require.Less(t, got, frameHeaderLen, "padding must be less than the whole header len")
		require.GreaterOrEqual(t, got, 0, "padding must be positive")
		require.Equal(t, 0, (got+int(length))%frameHeaderLen, "padding plus length must be a multiple of header len")
	}
}

func TestWriteIndexFrame(t *testing.T) {
	// TestFrameCodecFuzz covers most of the bases for the actual header encoding
	// etc. This just needs to test the index encoding.
	var index [1024]uint32

	for i := range index {
		// Write offsets as if each record is exactly 64 bytes
		index[i] = uint32(i * 64)
	}

	buf := make([]byte, indexFrameSize(len(index)))

	err := writeIndexFrame(buf, index[:])
	require.NoError(t, err)

	//t.Log(index, buf)

	// Validate that the encoded index after the header is what we expect
	offset := frameHeaderLen
	for i := range index {
		got := binary.LittleEndian.Uint32(buf[offset:])
		require.Equal(t, uint32(i*64), got, "unexpected index value at offset %d", i)
		offset += 4
	}
}
