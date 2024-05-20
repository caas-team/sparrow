SHELL := /bin/bash
.DEFAULT_GOAL := dev

.PHONY: dev
dev:
	@go build -tags=viper_bind_struct -o .tmp/sparrow ./main.go && sudo setcap CAP_NET_RAW+ep .tmp/sparrow
	@./.tmp/sparrow run --config ./.tmp/sparrow.yaml