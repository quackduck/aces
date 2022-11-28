# Aces

[comment]: <> (**A**ny **C**haracter **E**ncoding **S**et)
Any Character Encoding Set

Aces is a command line utility that lets you encode any data to a character set of your choice.

<sub><sup>_Psst... it is also now a library that you can use for encoding and decoding and also writing and reading at a bit level! See documentation [here](https://pkg.go.dev/github.com/quackduck/aces)._</sub></sup>

For example, you could encode "Foo Bar" to a combination of these four characters: "HhAa", resulting in this ~~hilarious~~ sequence of laughs:
```text
hHhAhAaahAaaHAHHhHHAhAHhhaHA
```
With Aces installed, you can actually do that with:
```shell
$ echo -n "Foo Bar" | aces HhAa
hHhAhAaahAaaHAHHhHHAhAHhhaHA
```
This was the original use of Aces (it was called `ha`, increased data size by 4X and had no decoder)

If you're on macOS, you can even convert that output to speech:
```shell
echo -n "Matthew Stanciu" | aces HhAa | say
```

Make your own wacky encoding:
```shell
$ echo HELLO WORLD | aces "DORK BUM"
RRD RBO RKD M  DRBU MBRRRKD RDOR
```

With Aces, you can see the actual 0s and 1s of files:
```shell
aces 01 < $(which echo)
```
You can also write hex/octal/binary/your own format by hand:
```shell
echo C2A7 | aces -d 0123456789ABCDEF
echo .+=. | aces -d ./+= # try this!
```
Convert binary to hex:
```shell
echo 01001010 | aces -d 01 | aces 0123456789ABCDEF
```

_Also check out the examples!_
## Installing

### macOS or Linux with linuxbrew
```shell
brew install quackduck/tap/aces
```

### Other platforms
Head over to [releases](https://github.com/quackduck/aces/releases) and download the latest binary!

## Usage
```yaml
Aces - Encode in any character set

Usage:
   aces <charset>               - encode data from STDIN into <charset>
   aces -d/--decode <charset>   - decode data from STDIN from <charset>
   aces -h/--help               - print this help message

Aces reads from STDIN for your data and outputs the result to STDOUT. The charset length must be
a power of 2. While decoding, bytes not in the charset are ignored. Aces does not add any padding.
```
## Examples
```shell
echo hello world | aces "<>(){}[]" | aces --decode "<>(){}[]"      # basic usage
echo matthew stanciu | aces HhAa | say                             # make funny sounds (macOS)
aces " X" < /bin/echo                                              # see binaries visually
echo 0100100100100001 | aces -d 01 | aces 01234567                 # convert bases
echo Calculus | aces 01                                            # what's stuff in binary?
echo Aces™ | base64 | aces -d
ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/   # even decode base64
```

## How does it work?
To answer that, we need to know how encoding works in general. Let's take the example of Base64.

### Base64
```text
ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/
```
That is the Base64 character set. As you may expect, it's 64 characters long. 

Let's say we want to somehow represent these two bytes in those 64 characters:
```text
00001001 10010010    # 09 92 in hex
```
To do that, Base64 does something very smart: it uses the bits, interpreted as a number, as indexes of the character set.

To explain what that means, let's consider what possible values 6 bits can represent: `000000` (decimal 0) to `111111` (decimal 63).
Since 0 to 63 is the exact range of indices that can be used with the 64 element character set, we'll group our 8 bit chunks (bytes) of data in 6 bit chunks (to use as indices):
```text
000010 011001 0010
```
`000010` is 2 in decimal, so by using it as an index of the character set, Base64 adds `C` (index 2) to the result.

`011001` is 16 + 8 + 1 = 25 in decimal, so Base64 appends `Z` (index 25) to the result.

You may have spotted a problem with the next chunk - it's only 4 bits long!

To get around this, Base64 pretends it's a 6 bit chunk and simply appends how many zeros are needed:
```
0010 + 00 => 001000
```
`001000` is 8 in decimal, so Base64 appends `I` to the result

But then, on the decoding side, how do you know where real data ends and where the pretend data starts?

It turns out that we don't need to do anything. On the decoding side, we know that the decoded data _has_ to be a multiple of 8 bits. So, the decoder ignores the bits which make the output _not_ a multiple of 8 bits, which will always be the extra bits we added.

Finally, encoding `00001001 10010010` to Base64 should result in `CZI`

Try this in your terminal with the real Base64!
```shell
echo -n -e \\x09\\x92 | base64 # base64 also adds a "=" character called "padding" to fit to a standard input length to output length ratio
```

### Aces

Now we generalize this to all character sets.

Generalizing the character set is easy, we just switch out the characters of the array storing the character set.

Changing the length of the character set is slightly harder. For every character set length, we need to figure out how many bits the chunked data should have. 

In the Base64 example, the chunk length (let's call it that) was 6. The character set length was 64.

[comment]: <> (Let's do another example: in octal, the character set length is 8 and the chunk length will be 3 &#40;`000` to `111` = 0 to 7&#41;)

[comment]: <> (For a character set length of 4, we'd need a chunk length of 2 &#40;`00` to `11` is 0 to 3&#41;)

[comment]: <> (```text)

[comment]: <> (set len => chunk len)

[comment]: <> (     4  => 2)

[comment]: <> (     8  => 3)

[comment]: <> (     64 => 6)

[comment]: <> (```)
It looks like `2^(chunk len) = set len`. We can prove this is true with this observation:

Every bit can either be 1 or 0, so the total possible values of a certain number of bits will just be `2^(number of bits)` (if you need further proof, observe that every bit we add doubles the total possibilities since there's an additional choice: the new bit being 0 or the new bit being 1)

The total possible values is the length of the character set (of course, since we need the indices to cover all the characters of the set)

So, to find the number of bits the chunked data should have, we just do `log2(character set length)`. Then, we divide the bytes into chunks of that many bits (which was pretty hard to implement: knowing when to read more bytes, crossing over into the next byte to fetch more bits, etc, etc.), use those bits as indices for the user-supplied character set, and print the result. Easy! (Nope, this is the work of several showers and a lot of late night pondering :) 






