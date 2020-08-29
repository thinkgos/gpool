#!/bin/bash
revive -config revive.toml  -formatter friendly ./...
golint ./... | reviewdog -f=golint -diff="git diff master"
golangci-lint run --out-format=line-number -E goimports -E misspell -E godot -E lll | reviewdog -f=golangci-lint -diff="git diff master"
