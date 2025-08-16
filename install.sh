#!/bin/bash

install_client() {
    mkdir -p build
    go build -o build/pvevdi
    cp build/pvevdi /usr/bin/pvevdi
}

if [[ $1 == "client" ]]; then
    install_client
else
    echo "Unknown or missing paramater"
    exit 1
fi
