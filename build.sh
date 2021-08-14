#!/usr/bin/env bash
go install
if [ "$(echo "heyheya " | aces 0123456789ABCDEF)" = "68657968657961200A" ]; then
   echo passed hex conversion
else 
   echo you stupid idiot the test failed
fi

if diff <(aces 0123456789ABCDEF < "$(which bash)" | aces -d 0123456789ABCDEF) "$(which bash)"; then
   echo whoa man, even the huge binary check worked
else
   echo you stupid idiot the binary check failed
fi
