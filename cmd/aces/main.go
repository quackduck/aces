package main

import (
	"fmt"
	"io"
	"math"
	"os"

	"github.com/quackduck/aces"
)

var (
	encodeHaHa []rune
	numOfBits  = 0
	decode     bool

	helpMsg = `Aces - Encode in any character set

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
		bw := aces.NewBitWriter(numOfBits, os.Stdout)
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
						err := bw.Write(byte(i))
						if err != nil {
							panic(err)
							return
						}
					}
				}
			}
		}
		bw.Flush()
		return
	}

	bs, err := aces.NewBitReader(numOfBits, os.Stdin)
	if err != nil {
		panic(err)
	}
	res := make([]byte, 0, 10*1024)
	for {
		chunk, err := bs.Read()
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
