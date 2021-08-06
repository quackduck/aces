package main

import (
	"fmt"
	"io"
	"math"
	"os"
)

var (
	encodeHaHa []rune
	//padding    = "="
	numOfBits = 0
	decode    bool
	//mustPadError = errors.New("bro turns out we need padding")
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
		//fmt.Printf("% 08b", chunk)
		if err != nil {
			if err == io.EOF {
				os.Stdout.Write(res)
				return
			}
			//if v, ok := err.(mustPadError); ok {
			//	os.Stdout.Write([]byte(strings.Repeat(padding, v.numOfPadding)))
			//}
			panic(err)
		}
		res = append(res, []byte(string(encodeHaHa[chunk]))...)
		if len(res) > 1024 {
			os.Stdout.Write(res)
			res = make([]byte, 0, 2*1024)
		}
	}
}

//type mustPadError struct {
//	numOfPadding int
//}
//
//func (e mustPadError) Error() string {
//	return "need this much padding: " + strconv.Itoa(e.numOfPadding)
//}

//// end bit is not included (just like slicing)
////   sliceByte(11001111, 2, 4) => delete first 2 bits => 00001111 => right shift by (8-4) => 00000000
////   sliceByte(11001111, 1, 7) => delete first 1 bit  => 01001111 => right shift by (8-7) => 00100111
//func sliceByte(b byte, start int, end int) byte {
//	return (b << start) >> (start+8-end)
//}

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
	//defer func() {
	//	if e != io.EOF {
	//		return
	//	}
	//	fmt.Printf("the end\n% 09b\n%s\nbitIdx = %d bufN = %d\n\n\n", bs.buf[bs.bitIdx/8], strings.Repeat(" ", bs.bitIdx % 8 + 1)+"^", bs.bitIdx, bs.bufN)
	//	//debug.PrintStack()
	//}()
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
				eh[0] = 0 // let it read from null byte (size can be inferred automatically at decoder)
				//bs.sendEOF = true
				bs.bufN-- // next call should simpy exit so we make it as if there isn't any more data (which is actually already true)
				//return 0, err
				//return 0, mustPadError{numOfPadding: bs.chunkLen - (8-(bs.bitIdx % 8))} // how many bytes we need from the next byte
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
		result = (firstByte << byte(bs.chunkLen-(8-(bs.bitIdx%8)))) + sliceByteLen(nextByte, 0, bs.chunkLen-(8-(bs.bitIdx%8)))
		if didChange {
			bs.bitIdx += bs.chunkLen - (8 - (bs.bitIdx % 8))
		}
	} else {
		result = sliceByteLen(bs.buf[bs.bitIdx/8], bs.bitIdx%8, bs.chunkLen)
	}
	//fmt.Printf("% 09b\n%s\nbitIdx = %d bufN = %d result =% 09b\n\n\n", bs.buf[bs.bitIdx/8], strings.Repeat(" ", bs.bitIdx % 8 + 1)+"^", bs.bitIdx, bs.bufN, result)
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
	//fmt.Fprintf(os.Stderr, "byte: % 09b, len: %d, bitNum: %d, byteNum: %d\n", b, bLen, bw.bitIdx%8, bw.bitIdx/8)
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
		//fmt.Printf("curr byte before      : % 09b\n", bw.buf[byteNum])
		bw.buf[byteNum] = bw.buf[byteNum] + (b << (8 - bitNum - bLen))
		//fmt.Printf("curr byte after       : % 09b\n", bw.buf[byteNum])
	} else {
		//fmt.Fprintf(os.Stderr, "curr byte before (spl): % 09b\n", bw.buf[byteNum])
		bw.buf[byteNum] = bw.buf[byteNum] + sliceByteLen(b, 8-bLen, 8-bitNum)
		//fmt.Fprintf(os.Stderr, "curr byte after (spl) : % 09b\n", bw.buf[byteNum])
		if len(bw.buf) <= byteNum+1 {
			_, err := bw.out.Write(bw.buf[:byteNum+1])
			if err != nil {
				return err
			}
			bw.init()
			//bw.buf[0] = sliceByteLen(b, 8-bLen+8-bitNum, 8-(8-bLen+8-bitNum)) << byte(8-(8-(8-bLen+8-bitNum)))
			//fmt.Fprintf(os.Stderr, "0th byte before       : % 09b\n", bw.buf[0])
			bw.buf[0] = sliceByteLen(b, 8-bLen+8-bitNum, 8-(8-bLen+8-bitNum)) << byte(8-bLen+8-bitNum)
			//fmt.Fprintf(os.Stderr, "0th byte after        : % 09b\n", bw.buf[0])
			bw.bitIdx = 0
			byteNum = 0
			bitNum = 0
			//panic(fmt.Sprintln("whooaaaaaaaaa my god byte:", byteNum))
		} else {
			//fmt.Printf("next byte before      : % 09b\n", bw.buf[byteNum+1])
			bw.buf[byteNum+1] = sliceByteLen(b, 8-bLen+8-bitNum, 8-(8-bLen+8-bitNum)) << byte(8-bLen+8-bitNum)
			//fmt.Printf("next byte after       : % 09b\n", bw.buf[byteNum+1])
		}
	}
	bw.bitIdx += bLen
	return nil
}

// call this only at the end
func (bw *bitWriter) flush() error {
	//fmt.Printf("Last byte of flush: % 09b", bw.buf[:bw.bitIdx/8][len(bw.buf[:bw.bitIdx/8])-1])
	_, err := bw.out.Write(bw.buf[:bw.bitIdx/8])
	return err
}

//	if 1 << numOfBits != len(encodeHaHa) {
//		panic("wrong charset length bruh, len(charset) should be a power of 2: "+ fmt.Sprint(encodeHaHa))
//	}
//	if 8 % numOfBits != 0 {
//		panic("wrong charset length bruh, log2(len(charset)) should be a factor of 8: "+ fmt.Sprint(encodeHaHa))
//	}
//	buf := make([]byte, 1024)
//	res := ""
//	for {
//		n, err := os.Stdin.Read(buf)
//		if err == io.EOF {
//			return
//		}
//		if err != nil {
//			fmt.Fprintln(os.Stderr, err)
//			return
//		}
//		for bufi := range buf {
//			if bufi > n {
//				goto Print
//			}
//			for bitMoveN := 0; bitMoveN < 8; bitMoveN +=numOfBits {
//				res += string(encodeHaHa[sliceByteLen(buf[bufi], bitMoveN, numOfBits)])
//			}
//		}
//Print:
//		os.Stdout.Write([]byte(res))
//	}

//
