#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
output="$root/dist/apps"

rm -rf "$output"
mkdir -p "$output"

for app in "$root"/apps/*; do
  if [ -d "$app" ]; then
    name=$(basename "$app")
    mkdir -p "$output/$name"
    cp -R "$app"/* "$output/$name"/
  fi
done

echo "Static apps copied to dist/apps"
