package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/quackduck/aces"
)

var (
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

Set $ACES_BUFSIZE to change the buffer size. The default is ` + strconv.Itoa(aces.BufSize) + ` bytes.

File issues, contribute or star at github.com/quackduck/aces`
)

func main() {
	var charset []rune
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "error: need at least one argument\n"+helpMsg)
		os.Exit(1)
	}
	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Println(helpMsg)
		return
	}
	decode := os.Args[1] == "--decode" || os.Args[1] == "-d"
	if decode {
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, "error: need character set\n"+helpMsg)
			os.Exit(1)
		}
		charset = []rune(os.Args[2])
	} else {
		charset = []rune(os.Args[1])
	}

	bufSizeString := os.Getenv("ACES_BUFSIZE")
	if bufSizeString != "" {
		bufSize, err := strconv.Atoi(bufSizeString)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: invalid buffer size:", err)
			return
		}
		aces.BufSize = bufSize
	}

	c, err := aces.NewCoding(charset)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}

	if decode {
		err := c.Decode(os.Stdout, os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		return
	}

	err = c.Encode(os.Stdout, os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
}
