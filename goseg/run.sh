#!/bin/bash
if [ "$1" = "prod" ]; then
  go run main.go | jq -R '. as $line | try (fromjson) catch $line'
else
  go run main.go dev | jq -R '. as $line | try (fromjson) catch $line'
fi
