package aces

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
)

var BufSize int = 16 * 1024

// sliceByteLen slices the byte b such that the result has length len and starting bit start
func sliceByteLen(b byte, start uint8, len uint8) byte {
	return (b << start) >> (8 - len)
}

type BitReader struct {
	chunkLen uint8
	in       io.Reader

	buf     []byte
	bitIdx  uint8
	byteIdx int
	bufN    int // limited by read size which cannot exceed an int
}

func NewBitReader(chunkLen uint8, in io.Reader) (*BitReader, error) {
	// bufSize % chunkLen == 0 so that we never have to read across the buffer boundary
	bs := &BitReader{chunkLen: chunkLen, in: in, buf: make([]byte, BufSize-BufSize%int(chunkLen))}
	var err error
	bs.bufN, err = bs.in.Read(bs.buf)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (br *BitReader) Read() (byte, error) {
	if br.byteIdx >= br.bufN { // need to read more
		n, err := br.in.Read(br.buf)
		if err != nil {
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

type BitWriter struct {
	chunkLen uint8
	out      io.Writer

	buf     []byte
	bitIdx  uint8
	byteIdx int
}

func NewBitWriter(chunkLen uint8, out io.Writer) *BitWriter {
	//bufSize % chunkLen == 0 so that we never have to write across the buffer boundary
	return &BitWriter{chunkLen: chunkLen, out: out, buf: make([]byte, BufSize-BufSize%int(chunkLen))}
}

func (bw *BitWriter) Write(b byte) error {
	if bw.byteIdx >= len(bw.buf) {
		_, err := bw.out.Write(bw.buf)
		if err != nil {
			return err
		}
		//fmt.Println("wrote ", len(bw.buf), string(bw.buf))
		bw.buf = make([]byte, BufSize-BufSize%int(bw.chunkLen))
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

// Flush writes the rest of the buffer. Only call this at the end of the stream.
func (bw *BitWriter) Flush() error {
	_, err := bw.out.Write(bw.buf[:bw.byteIdx])
	return err
}

type Coding struct {
	charset   []rune
	numOfBits uint8
}

// NewCoding creates a new Coding with the given character set. The length of the character set must be a power of 2
// and must not contain duplicate runes.
func NewCoding(charset []rune) (*Coding, error) {
	numOfBits := uint8(math.Log2(float64(len(charset))))
	if 1<<numOfBits != len(charset) {
		numOfBits = uint8(math.Round(math.Log2(float64(len(charset)))))
		return nil, errors.New(
			fmt.Sprintln("charset length is not a power of two.\n   have:", len(charset),
				"\n   want: a power of 2 (nearest is", 1<<numOfBits, "which is", math.Abs(float64(len(charset)-1<<numOfBits)), "away)"),
		)
	}
	seen := make(map[rune]bool)
	for _, r := range charset {
		if seen[r] {
			return nil, errors.New("charset contains duplicates")
		}
		seen[r] = true
	}
	return &Coding{charset: charset, numOfBits: numOfBits}, nil
}

func (c *Coding) Encode(dst io.Writer, src io.Reader) error {
	bs, err := NewBitReader(c.numOfBits, src)
	if err != nil {
		panic(err)
	}
	buf := make([]rune, 0, 10*1024)
	var chunk byte
	for {
		chunk, err = bs.Read()
		if err != nil {
			if err == io.EOF {
				_, err = dst.Write([]byte(string(buf)))
				if err != nil {
					return err
				}
				return nil
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

func (c *Coding) Decode(dst io.Writer, src io.Reader) error {
	bw := NewBitWriter(c.numOfBits, dst)
	bufStdin := bufio.NewReaderSize(src, 10*1024)
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
		err = bw.Write(runeToByte[r])
		if err != nil {
			return err
		}
	}
	bw.Flush()
	return nil
}
