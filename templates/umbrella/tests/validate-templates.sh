#!/bin/bash

mkdir -p ./tests/generated
mkdir -p ./tests/generated.last

printf "\n%s\n" "Running `helm template` command to generate manifests..."
helm template -g . | kubectl slice  -o tests/generated

printf "\n%s\n" "Running `kubeval` to validate manifests..."
# Check if kubeconform is installed
if ! command -v kubeconform &> /dev/null
then
    echo "kubeconform could not be found. Please install it to run this script."
    exit 1
fi

printf "\n%s\n" "Running `helm template` command to generate manifests..."

# Validate YAML files
for file in ./tests/generated/*.yaml; do
    if [ -f "$file" ]; then
        echo "Validating $file..."
        kubeconform "$file"
    else
        echo "No YAML files found to validate."
    fi
done

printf "\n%s\n" "Tests Complete!"
rm -rf ./tests/generated.last/* || true
mv ./tests/generated/* ./tests/generated.last/
