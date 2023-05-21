#!/bin/env sh

export DISCORD_TOKEN="$(cat token)"
go build
./andromeda
