#!/bin/bash

Whitelisted () {
        if [ "$1" = "shutdown" ]; then
                echo $1
        fi

        if [ "$1" = "reboot" ]; then
                echo $1
        fi
}

while true; do eval "$(Whitelisted $(cat commands))"; done

