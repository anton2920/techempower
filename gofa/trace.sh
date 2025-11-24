#!/bin/sh

set -e

PROJECT=techempower
go build -o $PROJECT  -tags gofatrace .

WRK=$HOME/Projects/thirdparty/wrk/

./$PROJECT &
sleep 0.5
$WRK/wrk -t 1 -c 1 -d 5 --script $WRK/plaintext.lua http://localhost:7073/plaintext -- 16
killall $PROJECT
