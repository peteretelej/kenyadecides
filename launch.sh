#!/bin/bash
set -e 

go vet .

go build -o kenyadecides main.go

source creds.env

./kenyadecides

