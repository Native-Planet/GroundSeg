#!/bin/bash
go run main.go dev | jq -R '. as $line | try (fromjson) catch $line'
