# Antenora Linux — top-level build.
# `make all` builds the package manager, the Stage3 tarball and the ISO.
# Everything is compiled from source by Dante. No Arch, no mkarchiso.

SHELL := /bin/bash
VERSION ?= 1.0.0
DIST   := dist

.PHONY: all dante stage3 iso test vet clean

all: dante stage3 iso

## dante — build the package manager binary
dante:
	CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o $(DIST)/dante ./cmd/dante

## stage3 — assemble the bootstrap tarball
stage3: dante
	./scripts/stage3-build.sh $(VERSION) x86_64

## iso — build the bootable ISO from source (toolchain -> base -> kernel -> ISO)
iso: dante
	./scripts/iso-build.sh $(VERSION) x86_64

## toolchain — bootstrap the Antenora toolchain into a sysroot
toolchain:
	./scripts/toolchain.sh

## test — run the unit tests
test:
	go test ./...

## vet — static analysis
vet:
	go vet ./...

## clean — remove build artifacts
clean:
	rm -rf $(DIST)
