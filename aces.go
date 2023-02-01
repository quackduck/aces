package aces

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
)

// size of the buffers used by BitReader and BitWriter.
const defaultBufSize = 16 * 1024

// size of the byte chunk whose base is converted at a time when the length of the character
// set is not a power of 2. See NewCoding for more detail.
const defaultNonPow2ByteChunkSize = 8

// sliceByteLen slices the byte b such that the result has length len and starting bit start
func sliceByteLen(b byte, start uint8, len uint8) byte {
	return (b << start) >> (8 - len)
}

// BitReader reads a constant number of bits from an io.Reader
type BitReader struct {
	chunkLen uint8
	in       io.Reader

	buf     []byte
	bufSize int
	bitIdx  uint8
	byteIdx int
	bufN    int // limited by read size which cannot exceed an int
}

// NewBitReader returns a BitReader that reads chunkLen bits at a time from in.
func NewBitReader(chunkLen uint8, in io.Reader) (*BitReader, error) {
	return NewBitReaderSize(chunkLen, in, defaultBufSize)
}

// NewBitReaderSize is like NewBitReader but allows setting the internal buffer size
func NewBitReaderSize(chunkLen uint8, in io.Reader, bufSize int) (*BitReader, error) {
	// bufSize % chunkLen == 0 so that we never have to read across the buffer boundary
	br := &BitReader{chunkLen: chunkLen, in: in, bufSize: bufSize - bufSize%int(chunkLen)}
	br.buf = make([]byte, br.bufSize)
	var err error
	br.bufN, err = io.ReadFull(br.in, br.buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return br, nil
}

// Read returns the next chunkLen bits from the stream. If there is no more data to read, it returns io.EOF.
// For example, if chunkLen is 3 and the next 3 bits are 101, Read returns 5, nil.
func (br *BitReader) Read() (byte, error) {
	if br.byteIdx >= br.bufN { // need to read more
		n, err := io.ReadFull(br.in, br.buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			return 0, err
		}
		br.byteIdx = 0
		br.bufN = n
	}

	var result byte
	if br.bitIdx+br.chunkLen > 8 { // want to slice past current byte
		firstByte := sliceByteLen(br.buf[br.byteIdx], br.bitIdx, 8-br.bitIdx)
		secondPartLen := br.chunkLen + br.bitIdx - 8
		result = (firstByte << secondPartLen) + sliceByteLen(br.buf[br.byteIdx+1], 0, secondPartLen)
	} else {
		result = sliceByteLen(br.buf[br.byteIdx], br.bitIdx, br.chunkLen)
	}
	br.bitIdx += br.chunkLen
	if br.bitIdx >= 8 {
		br.bitIdx %= 8
		br.byteIdx++
	}
	return result, nil
}

// BitWriter writes a constant number of bits to an io.Writer
type BitWriter struct {
	chunkLen uint8
	out      io.Writer

	buf     []byte
	bufSize int
	bitIdx  uint8
	byteIdx int
}

// NewBitWriter returns a BitWriter that writes chunkLen bits at a time to out.
func NewBitWriter(chunkLen uint8, out io.Writer) *BitWriter {
	return NewBitWriterSize(chunkLen, out, defaultBufSize)
}

// NewBitWriterSize is like NewBitWriter but allows setting the internal buffer size
func NewBitWriterSize(chunkLen uint8, out io.Writer, bufSize int) *BitWriter {
	//bufSize % chunkLen == 0 so that we never have to write across the buffer boundary
	bw := &BitWriter{chunkLen: chunkLen, out: out, bufSize: bufSize - bufSize%int(chunkLen)}
	bw.buf = make([]byte, bw.bufSize)
	return bw
}

// Write writes the last chunkLen bits from b to the stream.
// For example, if chunkLen is 3 and b is 00000101, Write writes 101.
func (bw *BitWriter) Write(b byte) error {
	if bw.byteIdx >= len(bw.buf) {
		_, err := bw.out.Write(bw.buf)
		if err != nil {
			return err
		}
		bw.buf = make([]byte, bw.bufSize)
		bw.bitIdx = 0
		bw.byteIdx = 0
	}

	if bw.bitIdx+bw.chunkLen > 8 { // write across byte boundary?
		// 8-bw.chunkLen is where b's actual data starts from.
		bStart := 8 - bw.chunkLen
		// space left in current byte
		left := 8 - bw.bitIdx

		bw.buf[bw.byteIdx] = bw.buf[bw.byteIdx] + sliceByteLen(b, bStart, left)
		// bStart + left is up to where b has been read from. (bw.chunkLen+br.bitIdx) - 8 is how many bits go to the next byte.
		bw.buf[bw.byteIdx+1] = sliceByteLen(b, bStart+left, bw.chunkLen-left) << (bStart + left) // simplified 8 - (bw.chunkLen + br.bitIdx - 8)
	} else {
		bw.buf[bw.byteIdx] = bw.buf[bw.byteIdx] + (b << (8 - (bw.bitIdx + bw.chunkLen)))
	}
	bw.bitIdx += bw.chunkLen
	if bw.bitIdx >= 8 {
		bw.bitIdx %= 8
		bw.byteIdx++
	}
	return nil
}

// Flush writes any remaining data in the buffer to the underlying io.Writer.
func (bw *BitWriter) Flush() error {
	_, err := bw.out.Write(bw.buf[:bw.byteIdx])
	return err
}

// Coding represents an encoding scheme for a character set. See NewCoding for more detail.
type Coding interface {
	// SetBufferSize sets internal buffer sizes
	SetBufferSize(size int)
	// SetByteChunkSize sets the number of bytes whose base is converted at time if the character set does not have a
	// length that is a power of 2. Encoding and decoding must be done with the same byte chunk size,
	SetByteChunkSize(size int)
	// Encode reads from src and encodes to dst
	Encode(dst io.Writer, src io.Reader) error
	// Decode reads from src and decodes to dst
	Decode(dst io.Writer, src io.Reader) error
}

// NewCoding creates a new coding with the given character set.
//
// For example,
//
//	NewCoding([]rune("0123456789abcdef"))
//
// creates a hex encoding scheme, and
//
//	NewCoding([]rune(" ❗"))
//
// creates a binary encoding scheme: 0s are represented by a space and 1s are represented by an exclamation mark.
//
// While a character set of any length can be used, those with power of 2 lengths (2, 4, 8, 16, 32, 64, 128, 256) use a
// more optimized algorithm.
//
// Sets that are not power of 2 in length use an algorithm that may not have the same output as other encoders with the
// same character set. For example, using the base58 character set does not mean that the output will be the same as a
// base58-specific encoder.
//
// This is because most encoders interpret data as a number and use a base conversion algorithm to convert it to the
// character set. For non-power-of-2 charsets, this requires all data to be read before encoding, which is not possible
// with streams. To enable stream encoding for non-power-of-2 charsets, Aces converts a default of 8 bytes (adjustable
// with Coding.SetByteChunkSize) of data at a time, which is not the same as converting the base of the entire data.
func NewCoding(charset []rune) (Coding, error) {
	seen := make(map[rune]bool)
	for _, r := range charset {
		if seen[r] {
			return nil, errors.New("charset contains duplicates: '" + string(r) + "'")
		}
		seen[r] = true
	}
	if len(charset)&(len(charset)-1) == 0 && len(charset) < 256 { // is power of 2?
		return newTwoCoding(charset)
	}
	return newAnyCoding(charset)
}

// twoCoding is for character sets of a length that is a power of 2.
type twoCoding struct {
	charset   []rune
	numOfBits uint8
	bufSize   int
}

func newTwoCoding(charset []rune) (*twoCoding, error) {
	numOfBits := uint8(math.Log2(float64(len(charset))))
	if 1<<numOfBits != len(charset) {
		numOfBits = uint8(math.Round(math.Log2(float64(len(charset)))))
		return nil, errors.New(
			fmt.Sprintln("charset length is not a power of two.\n   have:", len(charset),
				"\n   want: a power of 2 (nearest is", 1<<numOfBits, "which is", math.Abs(float64(len(charset)-1<<numOfBits)), "away)"),
		)
	}
	return &twoCoding{charset: charset, numOfBits: numOfBits}, nil
}

func (c *twoCoding) SetByteChunkSize(_ int)    {}
func (c *twoCoding) SetBufferSize(bufSize int) { c.bufSize = bufSize }

func (c *twoCoding) Encode(dst io.Writer, src io.Reader) error {
	bs, err := NewBitReaderSize(c.numOfBits, src, c.bufSize)
	if err != nil {
		panic(err)
	}
	buf := make([]rune, 0, c.bufSize)
	var chunk byte
	for {
		chunk, err = bs.Read()
		if err != nil {
			if err == io.EOF {
				_, err = dst.Write([]byte(string(buf)))
			}
			return err
		}
		buf = append(buf, c.charset[chunk])
		if len(buf) == cap(buf) {
			_, err = dst.Write([]byte(string(buf)))
			if err != nil {
				return err
			}
			buf = buf[:0]
		}
	}
}

func (c *twoCoding) Decode(dst io.Writer, src io.Reader) error {
	bw := NewBitWriterSize(c.numOfBits, dst, c.bufSize)
	bufStdin := bufio.NewReaderSize(src, c.bufSize)
	runeToByte := make(map[rune]byte, len(c.charset))
	for i, r := range c.charset {
		runeToByte[r] = byte(i)
	}
	for {
		r, _, err := bufStdin.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		b, ok := runeToByte[r]
		if !ok {
			if r == '\n' || r == '\r' {
				continue
			}
			return errors.New("character " + string(r) + "in input is not in the character set")
		}
		err = bw.Write(b)
		if err != nil {
			return err
		}
	}
	return bw.Flush()
}

// anyCoding works with character sets of any length but is less performant than twoCoding.
type anyCoding struct {
	charset   []rune
	chunkSize int
	rPerChunk int
	bufSize   int
}

func newAnyCoding(charset []rune) (*anyCoding, error) {
	return &anyCoding{charset, defaultNonPow2ByteChunkSize, runesPerChunk(charset, defaultNonPow2ByteChunkSize), defaultBufSize}, nil
}

//// newAnyCodingWithChunkSize allows setting the length of the chunk whose base is converted at a time
//func newAnyCodingWithChunkSize(charset []rune, chunkSize int) (*anyCoding, error) {
//	return &anyCoding{charset, chunkSize, runesPerChunk(charset, chunkSize), defaultBufSize}, nil
//}

func (c *anyCoding) SetByteChunkSize(size int) {
	c.chunkSize = size
	c.rPerChunk = runesPerChunk(c.charset, size)
}

func (c *anyCoding) SetBufferSize(bufSize int) { c.bufSize = bufSize }

func (c *anyCoding) Encode(dst io.Writer, src io.Reader) error {
	//br := bufio.NewReaderSize(src, c.bufSize)
	result := make([]rune, 0, c.bufSize)
	buf := make([]byte, c.chunkSize)
	for {
		_, err := io.ReadFull(src, buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			if err == io.EOF {
				_, err = dst.Write([]byte(string(result)))
			}
			return err
		}

		result = append(result, encodeByteChunk(c.charset, buf, c.rPerChunk)...)

		if len(result)+(8*c.chunkSize) > cap(result) { // (8*c.chunkSize) is the max size of the result (if charset is binary)
			_, err = dst.Write([]byte(string(result)))
			if err != nil {
				return err
			}
			result = result[:0]
		}
	}
}

var resultBuf = make([]rune, 0, 64)

func encodeByteChunk(set []rune, octet []byte, rPerChunk int) []rune {
	resultBuf = resultBuf[:0]
	i := bytesToInt(octet)
	resultBuf = toBase(i, resultBuf, set)
	for len(resultBuf) < rPerChunk {
		// prepend with minimum new allocations
		resultBuf = append(resultBuf, 0)
		copy(resultBuf[1:], resultBuf)
		resultBuf[0] = set[0]
	}
	return resultBuf
}

func decodeToByteChunk(set []rune, runes []rune, chunkSize int) ([]byte, error) {
	num, err := fromBase(runes, set)
	if err != nil {
		return nil, err
	}
	return num.FillBytes(make([]byte, chunkSize)), nil
}

func (c *anyCoding) Decode(dst io.Writer, src io.Reader) error {
	var err error

	br := bufio.NewReaderSize(src, c.bufSize)
	result := make([]byte, 0, c.bufSize)
	buf := make([]rune, c.rPerChunk)
	var chunk []byte
	for {
		for i := range buf {
			buf[i], _, err = br.ReadRune()
			if err != nil {
				if err == io.EOF {
					_, err = dst.Write(result)
				}
				println("Holy shit lol")
				return err
			}
			if buf[i] == '\n' || buf[i] == '\r' {
				i-- // ignore newline, read rune again
			}
		}

		chunk, err = decodeToByteChunk(c.charset, buf, c.chunkSize)
		if err != nil {
			return err
		}
		result = append(result, chunk...)

		if len(result)+c.chunkSize > cap(result) {
			_, err = dst.Write(result)
			if err != nil {
				return err
			}
			result = result[:0]
		}
	}
}

func runesPerChunk(set []rune, chunkLen int) int {
	return int(math.Ceil(float64(8*chunkLen) / math.Log2(float64(len(set)))))
}

func bytesToInt(b []byte) *big.Int {
	return (&big.Int{}).SetBytes(b)
}

func toBase(num *big.Int, buf []rune, set []rune) []rune {
	base := int64(len(set))
	div, rem := new(big.Int), new(big.Int)
	div.QuoRem(num, big.NewInt(base), rem)
	if div.Cmp(big.NewInt(0)) != 0 {
		buf = append(buf, toBase(div, buf, set)...)
	}
	return append(buf, set[rem.Uint64()])
}

func fromBase(enc []rune, set []rune) (*big.Int, error) {
	result := new(big.Int)
	setlen := len(set)

	setMap := make(map[rune]int64)
	for i, r := range set {
		setMap[r] = int64(i)
	}

	numOfDigits := len(enc)
	for i := 0; i < numOfDigits; i++ {
		mult := new(big.Int).Exp( // setlen ^ numOfDigits-i-1 = the "place value"
			big.NewInt(int64(setlen)),
			big.NewInt(int64(numOfDigits-i-1)),
			nil,
		)
		idx := setMap[enc[i]]
		if idx == -1 {
			return nil, errors.New("character " + string(enc[i]) + "in input is not in the character set")
		}
		mult.Mul(mult, big.NewInt(idx)) // multiply "place value" with the digit at spot i
		result.Add(result, mult)
	}
	return result, nil
}
