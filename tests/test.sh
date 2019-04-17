#!/bin/bash
set -x

go build github.com/b1naryth1ef/heracles/cmd/heracles && python -m pytest $@
