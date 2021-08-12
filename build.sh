#!/usr/bin/env bash
go install
if [ "$(echo "heyheya " | encodin 0123456789ABCDEF)" = "68657968657961200A" ]; then
   echo passed hex conversion
else 
   echo you stupid idiot the test failed
fi

if diff <(encodin 0123456789ABCDEF < "$(which devchat)" | encodin -d 0123456789ABCDEF) "$(which devchat)"; then
   echo whoa man, even the huge binary check worked
else
   echo you stupid idiot the binary check failed
fi
