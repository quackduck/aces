package main

import (
	"fmt"
	"io"
	"math"
	"os"
)

var (
	encodeHaHa []rune
	numOfBits  = 0
	decode     bool
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("yo i need at least one arg mate")
		os.Exit(1)
	}
	decode = os.Args[1] == "decode" || os.Args[1] == "d"
	if decode {
		encodeHaHa = []rune(os.Args[2])
	} else {
		encodeHaHa = []rune(os.Args[1])
	}
	numOfBits = int(math.Log2(float64(len(encodeHaHa))))
	if 1<<numOfBits != len(encodeHaHa) {
		fmt.Println("wrong charset length. have:", len(encodeHaHa), "want: a power of 2")
		os.Exit(1)
	}

	if decode {
		bw := bitWriter{chunkLen: numOfBits, out: os.Stdout}
		bw.init()
		buf := make([]byte, 10*1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}
			for _, c := range []rune(string(buf[:n])) {
				for i, char := range encodeHaHa {
					if c == char {
						err := bw.write(byte(i), numOfBits)
						if err != nil {
							panic(err)
							return
						}
					}
				}
			}
		}
		bw.flush()
		return
	}

	bs := bitStreamer{chunkLen: numOfBits, in: os.Stdin}
	err := bs.init()
	if err != nil {
		panic(err)
	}
	res := make([]byte, 0, 2*1024)
	for {
		chunk, err := bs.next()
		if err != nil {
			if err == io.EOF {
				os.Stdout.Write(res)
				return
			}
			panic(err)
		}
		res = append(res, []byte(string(encodeHaHa[chunk]))...)
		if len(res) > 1024 {
			os.Stdout.Write(res)
			res = make([]byte, 0, 2*1024)
		}
	}
}

// sliceByteLen slices the byte b such that the result has length len and starting bit start
func sliceByteLen(b byte, start int, len int) byte {
	return (b << start) >> byte(8-len)
	//return sliceByte(b, start, start+len)
}

type bitStreamer struct {
	// set these
	chunkLen int
	in       io.Reader

	// internal vars
	buf    []byte
	bitIdx int
	bufN   int
}

func (bs *bitStreamer) init() error {
	bs.buf = make([]byte, 16*1024)
	n, err := bs.in.Read(bs.buf)
	if err != nil {
		return err
	}
	bs.bufN = n
	return nil
}

func (bs *bitStreamer) next() (b byte, e error) {
	if bs.bitIdx/8 >= bs.bufN { // need to read more?
		n, err := bs.in.Read(bs.buf)
		if err != nil {
			return 0, err
		}
		bs.bitIdx = bs.bitIdx % 8
		bs.bufN = n
	}

	var result byte
	if bs.bitIdx%8+bs.chunkLen > 8 { // want to slice past current byte
		currByte := bs.buf[bs.bitIdx/8]
		didChange := false
		if bs.bitIdx/8+1 >= bs.bufN { // unlikely
			didChange = true
			eh := make([]byte, 1)
			_, err := bs.in.Read(eh) // the actual data size doesn't change so we won't change n
			if err != nil {
				eh[0] = 0 // let it read from null byte (size can be inferred automatically at decoder (result has to be multiples of 8 bits))
				bs.bufN-- // next call should simpy exit so we make it as if there isn't any more data (which is actually already true)
			}
			if bs.bitIdx/8+1 >= len(bs.buf) {
				bs.buf = append(bs.buf, eh[0])
			} else {
				bs.buf[bs.bitIdx/8+1] = eh[0]
			}
			bs.bufN++
		}
		nextByte := bs.buf[bs.bitIdx/8+1]

		firstByte := sliceByteLen(currByte, bs.bitIdx%8, 8-(bs.bitIdx%8))
		result = (firstByte << byte(bs.chunkLen+(bs.bitIdx%8)-8)) + sliceByteLen(nextByte, 0, bs.chunkLen+(bs.bitIdx%8)-8)
		if didChange {
			bs.bitIdx += bs.chunkLen - (8 - (bs.bitIdx % 8))
		}
	} else {
		result = sliceByteLen(bs.buf[bs.bitIdx/8], bs.bitIdx%8, bs.chunkLen)
	}
	bs.bitIdx += bs.chunkLen
	return result, nil
}

type bitWriter struct {
	chunkLen int
	out      io.Writer

	buf    []byte
	bitIdx int
}

func (bw *bitWriter) init() {
	bw.buf = make([]byte, 16*1024)
}

func (bw *bitWriter) write(b byte, bLen int) error {
	bitNum := bw.bitIdx % 8
	byteNum := bw.bitIdx / 8
	if byteNum >= len(bw.buf) {
		_, err := bw.out.Write(bw.buf)
		if err != nil {
			return err
		}
		bw.init()
		bw.bitIdx = 0
		bitNum = bw.bitIdx % 8
		byteNum = bw.bitIdx / 8
	}

	if 8-bitNum-bLen >= 0 {
		bw.buf[byteNum] = bw.buf[byteNum] + (b << (8 - bitNum - bLen))
	} else {
		bw.buf[byteNum] = bw.buf[byteNum] + sliceByteLen(b, 8-bLen, 8-bitNum)
		if len(bw.buf) <= byteNum+1 {
			_, err := bw.out.Write(bw.buf[:byteNum+1])
			if err != nil {
				return err
			}
			bw.init()
			bw.buf[0] = sliceByteLen(b, 8-bLen+8-bitNum, bLen+bitNum-8) << byte(8-bLen+8-bitNum)
			bw.bitIdx = 0
			byteNum = 0
			bitNum = 0
		} else {
			bw.buf[byteNum+1] = sliceByteLen(b, 8-bLen+8-bitNum, bLen+bitNum-8) << byte(8-bLen+8-bitNum)
		}
	}
	bw.bitIdx += bLen
	return nil
}

// call this only at the end
func (bw *bitWriter) flush() error {
	_, err := bw.out.Write(bw.buf[:bw.bitIdx/8])
	return err
}
