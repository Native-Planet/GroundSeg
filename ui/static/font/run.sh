#!/bin/bash

# Loop through each file in the current directory
for file in *; do
    # Check if the filename contains uppercase characters
    if [[ $file =~ [A-Z] ]]; then
        # Convert filename to lowercase and rename the file
        mv "$file" "`echo $file | tr '[:upper:]' '[:lower:]'`"
    fi
done

