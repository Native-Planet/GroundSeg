#!/bin/bash
go run main.go dev | jq -R '. as $line | try (fromjson) catch $line'
#go run main.go | jq -R '. as $line | try (fromjson) catch $line'
