// can't believe this code actually works lol

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
	bufSize    = 16 * 1024
	helpMsg    = `Aces - Encode in any character set

Usage:
   aces <charset>               - encode data from STDIN into <charset>
   aces -d/--decode <charset>   - decode data from STDIN from <charset>
   aces -h/--help               - print this help message

Aces reads from STDIN for your data and outputs the result to STDOUT. The charset length must be
a power of 2. While decoding, bytes not in the charset are ignored. Aces does not add any padding.

Examples:
   echo hello world | aces "<>(){}[]" | aces --decode "<>(){}[]"      # basic usage
   echo matthew stanciu | aces HhAa | say                             # make funny sounds (macOS)
   aces " X" < /bin/echo                                              # see binaries visually
   echo 0100100100100001 | aces -d 01 | aces 01234567                 # convert bases
   echo Calculus | aces 01                                            # what's stuff in binary?
   echo Aces™ | base64 | aces -d
   ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/   # even decode base64

File issues, contribute or star at github.com/quackduck/aces`
)

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "error: need at least one argument\n"+helpMsg)
		os.Exit(1)
	}
	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Println(helpMsg)
		return
	}
	decode = os.Args[1] == "--decode" || os.Args[1] == "-d"
	if decode {
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, "error: need character set\n"+helpMsg)
			os.Exit(1)
		}
		encodeHaHa = []rune(os.Args[2])
	} else {
		encodeHaHa = []rune(os.Args[1])
	}
	numOfBits = int(math.Log2(float64(len(encodeHaHa))))
	if 1<<numOfBits != len(encodeHaHa) {
		numOfBits = int(math.Round(math.Log2(float64(len(encodeHaHa)))))
		fmt.Fprintln(os.Stderr, "error: charset length is not a power of two.\n   have:", len(encodeHaHa), "\n   want: a power of 2 (nearest is", 1<<numOfBits, "which is", math.Abs(float64(len(encodeHaHa)-1<<numOfBits)), "away)")
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
						err := bw.write(byte(i))
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
				os.Stdout.Close()
				return
			}
			panic(err)
		}
		res = append(res, string(encodeHaHa[chunk])...)
		if len(res) > 1024*7/2 {
			os.Stdout.Write(res)
			res = make([]byte, 0, 2*1024)
		}
	}
}

// sliceByteLen slices the byte b such that the result has length len and starting bit start
func sliceByteLen(b byte, start int, len int) byte {
	return (b << start) >> byte(8-len)
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
	//bufSize % bs.chunkLen == 0 so that we never have to read across the buffer boundary
	bs.buf = make([]byte, bufSize-bufSize%bs.chunkLen)
	var err error
	bs.bufN, err = bs.in.Read(bs.buf)
	if err != nil {
		return err
	}
	return nil
}

func (bs *bitStreamer) next() (byte, error) {
	byteNum := bs.bitIdx / 8
	bitNum := bs.bitIdx % 8
	if byteNum >= bs.bufN { // need to read more
		n, err := bs.in.Read(bs.buf)
		if err != nil {
			return 0, err
		}
		bs.bitIdx = bitNum
		byteNum = bs.bitIdx / 8
		bitNum = bs.bitIdx % 8
		bs.bufN = n
	}

	var result byte
	if bitNum+bs.chunkLen > 8 { // want to slice past current byte
		currByte := bs.buf[byteNum]
		firstByte := sliceByteLen(currByte, bitNum, 8-bitNum)
		var nextByte byte
		nextByte = bs.buf[byteNum+1]
		secondByteLen := bs.chunkLen + bitNum - 8
		result = (firstByte << byte(secondByteLen)) + sliceByteLen(nextByte, 0, secondByteLen)
		bs.bitIdx += bs.chunkLen
		return result, nil
	}
	result = sliceByteLen(bs.buf[byteNum], bitNum, bs.chunkLen)
	bs.bitIdx += bs.chunkLen
	return result, nil
}

func errPrint(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
}

type bitWriter struct {
	chunkLen int
	out      io.Writer

	buf    []byte
	bitIdx int
}

func (bw *bitWriter) init() {
	//bufSize % bw.chunkLen == 0 so that we never have to write across the buffer boundary
	bw.buf = make([]byte, bufSize-bufSize%bw.chunkLen)
}

func (bw *bitWriter) write(b byte) error {
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

	if bitNum+bw.chunkLen > 8 {
		bw.buf[byteNum] = bw.buf[byteNum] + sliceByteLen(b, 8-bw.chunkLen, 8-bitNum)
		bw.buf[byteNum+1] = sliceByteLen(b, 8-bw.chunkLen+8-bitNum, bw.chunkLen+bitNum-8) << byte(8-bw.chunkLen+8-bitNum)
	} else {
		bw.buf[byteNum] = bw.buf[byteNum] + (b << (8 - (bitNum + bw.chunkLen)))
	}
	bw.bitIdx += bw.chunkLen
	return nil
}

// call this only at the end
func (bw *bitWriter) flush() error {
	_, err := bw.out.Write(bw.buf[:bw.bitIdx/8])
	return err
}
