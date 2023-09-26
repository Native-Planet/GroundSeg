#!/bin/bash
cd ui
/bin/bash build.sh
cd ../goseg
go run main.go dev