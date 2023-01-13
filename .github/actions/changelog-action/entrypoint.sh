#!/bin/bash

filename="CHANGELOG.md"
printing=false
#While loop to read line by line
while IFS= read -r line; do
  #If the line starts with ## & currently printing, disable printing
  if [[ $line == \#\#* ]] && [[ $printing == true ]]; then
    break
  fi
  # If printing is enabled, print the line.
  if [[ $printing == true ]]; then
    echo "$line"
  fi
  #If the line starts with ## & not currently printing, enable printing
  if [[ $line == "## [$VERSION]"* ]] && [[ $printing == false ]]; then
    printing=true
  fi
done <"$filename"
