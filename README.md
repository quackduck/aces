# Aces

[comment]: <> (**A**ny **C**haracter **E**ncoding **S**et)
Any Character Encoding Set

Aces is a command line utility that lets you encode any file to a character set of your choice.

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
$ echo -n "Matthew Stanciu" | aces HhAa | say
```

With Aces, you can see the actual 0s and 1s of files:
```shell
$ aces 01 < $(which echo)
```
You can also write hex/octal/binary/your own format by hand:
```shell
$ echo C2A7 | aces -d 0123456789ABCDEF
$ echo .+=. | aces -d ./+= # try this!
```

## Installing

### macOS or Linux with linuxbrew
```shell
brew install quackduck/tap/aces
```

### Other platforms
Head over to [releases](https://github.com/quackduck/aces/releases) and download the latest binary!