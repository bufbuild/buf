#!/usr/bin/env bash

set -e

# This is just a primitive diff script for capturing the output
cbuf_out=$(mktemp /tmp/cbufout.XXXXXXX)
buf_out=$(mktemp /tmp/bufout.XXXXXXX)
./connectbuf/cbuf $@ 2>&1 | tee $cbuf_out > /dev/null
buf  $@ 2>&1 | tee $buf_out > /dev/null
output=$(mktemp /tmp/out.XXXXXX)
diff $buf_out $cbuf_out 2>&1 | tee $output > /dev/null
if [ -s $output ]; then
  echo "~~~~~~~~~~"
  cat $output
  echo "~~~~~~~~~~"
fi

rm $buf_out
rm $cbuf_out
