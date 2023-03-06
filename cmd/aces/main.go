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
   aces <charset>                  - encode data from STDIN into <charset>
   aces -d/--decode <charset>      - decode data from STDIN from <charset>
   aces -v/--version | -h/--help   - print version or this help message

Aces reads from STDIN for your data and outputs the result to STDOUT. An optimized algorithm is used 
for character sets with a power of 2 length. Newlines are ignored when decoding.

Examples:
   echo hello world | aces "<>(){}[]" | aces --decode "<>(){}[]"      # basic usage
   echo matthew stanciu | aces HhAa | say                             # make funny sounds (macOS)
   aces " X" < /bin/echo                                              # see binaries visually
   echo 0100100100100001 | aces -d 01 | aces 0123456789abcdef         # convert bases
   echo Calculus | aces 01                                            # what's stuff in binary?
   echo Aces™ | base64 | aces -d
   ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/   # even decode base64
   echo -n uwonsmth | aces 🥇🥈🥉                                    # emojis work too! 
Set the encoding/decoding buffer size with --bufsize <size> (default 16KiB).

File issues, contribute or star at github.com/quackduck/aces`
	version = "dev"
)

func main() {
	var charset []rune
	var err error
	bufsize := 16 * 1024

	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "error: need at least one argument\n"+helpMsg)
		return
	}

	if os.Args[1] == "--bufsize" {
		if len(os.Args)+1 < 2 { // index 2 available?
			fmt.Fprintln(os.Stderr, "error: need a value for --bufsize\n"+helpMsg)
			return
		}
		bufsize, err = strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: invalid value for --bufsize\n"+helpMsg)
			return
		}
		os.Args = os.Args[2:]
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Println(helpMsg)
		return
	}

	if os.Args[1] == "-v" || os.Args[1] == "--version" {
		fmt.Println("Aces " + version)
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

	c, err := aces.NewCoding(charset)
	c.SetBufferSize(bufsize)
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
