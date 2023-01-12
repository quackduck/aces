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
   echo 0100100100100001 | aces -d 01 | aces 0123456789abcdef         # convert bases
   echo Calculus | aces 01                                            # what's stuff in binary?
   echo Aces™ | base64 | aces -d
   ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/   # even decode base64

Set the encoding/decoding buffer size with --bufsize <size> (default ` + strconv.Itoa(aces.BufSize) + ` bytes).

File issues, contribute or star at github.com/quackduck/aces`
)

func main() {
	var charset []rune

	//i := aces.ImpureCoding{}
	//i.Charset = []rune("012")
	//err := i.Encode(os.Stdout, os.Stdin)
	//if err != nil {
	//	fmt.Fprintln(os.Stderr, "error:", err)
	//}
	//return

	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "error: need at least one argument\n"+helpMsg)
		return
	}

	if os.Args[1] == "--bufsize" {
		if len(os.Args)+1 < 2 { // index 2 avaliable?
			fmt.Fprintln(os.Stderr, "error: need a value for --bufsize\n"+helpMsg)
			return
		}
		bufsize, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: invalid value for --bufsize\n"+helpMsg)
			return
		}
		aces.BufSize = bufsize
		os.Args = os.Args[2:]
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Println(helpMsg)
		return
	}
	decode := os.Args[1] == "--decode" || os.Args[1] == "-d"
	if decode {
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, "error: need character set\n"+helpMsg)
			return
		}
		charset = []rune(os.Args[2])
	} else {
		charset = []rune(os.Args[1])
	}

	// check if charset length isn't a power of 2
	if len(charset)&(len(charset)-1) != 0 {
		c, err := aces.NewImpureCoding(charset)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return
		}
		if decode {
			err = c.Decode(os.Stdout, os.Stdin)
		} else {
			err = c.Encode(os.Stdout, os.Stdin)
			fmt.Println()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		return
	}

	c, err := aces.NewCoding(charset)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}

	if decode {
		err = c.Decode(os.Stdout, os.Stdin)
	} else {
		err = c.Encode(os.Stdout, os.Stdin)
		fmt.Println()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
}
