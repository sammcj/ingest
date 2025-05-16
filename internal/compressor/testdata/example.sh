#!/bin/bash

# This is a bash comment

# Define a function
function greet() {
    local name=$1
    echo "Hello, $name!"
    if [[ ${#name} -gt 3 ]]; then
        echo "That's a long name!"
    fi
    return 0
}

# Define another function
process_file() {
    local file=$1
    if [[ -f "$file" ]]; then
        echo "Processing $file..."
        cat "$file" | grep "pattern"
    else
        echo "File not found: $file"
        return 1
    fi
}

# Main script execution
echo "Starting script"
greet "World"
process_file "/tmp/example.txt"

# Conditional logic
if [[ $? -eq 0 ]]; then
    echo "Success!"
else
    echo "Failed!"
fi

# Loop example
for i in {1..5}; do
    echo "Iteration $i"
done

exit 0
