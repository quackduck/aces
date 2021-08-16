#!/usr/bin/env bash
set -e
go install
#if [ "$(echo "heyheya " | aces 0123456789ABCDEF)" = "68657968657961200A" ]; then
#   echo passed hex conversion
#else
#   echo you stupid idiot the test failed
#fi

[ "$(echo "heyheya " | aces 0123456789ABCDEF)" = "68657968657961200A" ] || echo HEX CHECK FAILED, slap the programmer

#if diff <(aces 0123456789ABCDEF < "$(which bash)" | aces -d 0123456789ABCDEF) "$(which bash)"; then
#   echo whoa man, even the huge binary check worked
#else
#   echo you stupid idiot the binary check failed
#fi

diff <(aces 0123456789ABCDEF < "$(which bash)" | aces -d 0123456789ABCDEF) "$(which bash)" || echo big binary check failed, slap the programmer

echo hello world | aces "<>(){}[]" | aces --decode "<>(){}[]" > /dev/null || echo example 1 failed
echo matthew stanciu | aces HhAa > /dev/null || echo example 2 failed
aces " X" < /bin/echo > /dev/null || echo example 3 failed
echo 0100100100100001 | aces -d 01 | aces 01234567 > /dev/null || echo example 4 failed
echo Calculus | aces 01 > /dev/null || echo example 5 failed
echo "Aces™" | base64 | aces -d ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/ > /dev/null || echo example 6 failed
