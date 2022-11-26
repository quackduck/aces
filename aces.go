package aces

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
)

var BufSize = 16 * 1024

// sliceByteLen slices the byte b such that the result has length len and starting bit start
func sliceByteLen(b byte, start int, len int) byte {
	return (b << start) >> byte(8-len)
}

type BitReader struct {
	chunkLen int
	in       io.Reader

	buf    []byte
	bitIdx int
	bufN   int
}

func NewBitReader(chunkLen int, in io.Reader) (*BitReader, error) {
	//bufSize % bs.chunkLen == 0 so that we never have to read across the buffer boundary
	bs := &BitReader{chunkLen: chunkLen, in: in, buf: make([]byte, BufSize-BufSize%chunkLen)}
	var err error
	bs.bufN, err = bs.in.Read(bs.buf)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (br *BitReader) Read() (byte, error) {
	byteNum := br.bitIdx / 8
	bitNum := br.bitIdx % 8
	if byteNum >= br.bufN { // need to read more
		n, err := br.in.Read(br.buf)
		if err != nil {
			return 0, err
		}
		br.bitIdx = bitNum
		byteNum = br.bitIdx / 8
		bitNum = br.bitIdx % 8
		br.bufN = n
	}

	var result byte
	if bitNum+br.chunkLen > 8 { // want to slice past current byte
		firstByte := sliceByteLen(br.buf[byteNum], bitNum, 8-bitNum)
		secondPartLen := br.chunkLen + bitNum - 8
		result = (firstByte << secondPartLen) + sliceByteLen(br.buf[byteNum+1], 0, secondPartLen)
		br.bitIdx += br.chunkLen
		return result, nil
	}
	result = sliceByteLen(br.buf[byteNum], bitNum, br.chunkLen)
	br.bitIdx += br.chunkLen
	return result, nil
}

type BitWriter struct {
	chunkLen int
	out      io.Writer

	buf    []byte
	bitIdx int
}

func NewBitWriter(chunkLen int, out io.Writer) *BitWriter {
	//bufSize % bw.chunkLen == 0 so that we never have to write across the buffer boundary
	return &BitWriter{chunkLen: chunkLen, out: out, buf: make([]byte, BufSize-BufSize%chunkLen)}
}

func (bw *BitWriter) Write(b byte) error {
	bitNum := bw.bitIdx % 8
	byteNum := bw.bitIdx / 8
	if byteNum >= len(bw.buf) {
		_, err := bw.out.Write(bw.buf)
		if err != nil {
			return err
		}
		bw.buf = make([]byte, BufSize-BufSize%bw.chunkLen)
		bw.bitIdx = 0
		bitNum = 0
		byteNum = 0
	}

	if bitNum+bw.chunkLen > 8 { // write across byte boundary?
		// 8-bw.chunkLen is where b's actual data starts from.
		bStart := 8 - bw.chunkLen
		// space left in current byte
		left := 8 - bitNum

		bw.buf[byteNum] = bw.buf[byteNum] + sliceByteLen(b, bStart, left)
		// bStart + left is up to where b has been read from. (bw.chunkLen+bitNum) - 8 is how many bits go to the next byte.
		bw.buf[byteNum+1] = sliceByteLen(b, bStart+left, bw.chunkLen-left) << (bStart + left) // simplified 8 - (bw.chunkLen + bitNum - 8)
	} else {
		bw.buf[byteNum] = bw.buf[byteNum] + (b << (8 - (bitNum + bw.chunkLen)))
	}
	bw.bitIdx += bw.chunkLen
	return nil
}

// Flush writes the rest of the buffer. Only call this at the end of the stream.
func (bw *BitWriter) Flush() error {
	_, err := bw.out.Write(bw.buf[:bw.bitIdx/8])
	return err
}

type Coding struct {
	charset   []rune
	numOfBits int
}

// NewCoding creates a new Coding with the given character set. The length of the character set must be a power of 2
// and must not contain duplicate runes.
func NewCoding(charset []rune) (*Coding, error) {
	numOfBits := int(math.Log2(float64(len(charset))))
	if 1<<numOfBits != len(charset) {
		numOfBits = int(math.Round(math.Log2(float64(len(charset)))))
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
