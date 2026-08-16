# Build the Antenora live ISO via mkarchiso.
# Usage: make iso VERSION=1.0.0

VERSION ?= 1.0.0
PROFILE := archiso
OUT     := ../dist/iso

.PHONY: all iso clean

all: iso

iso: dante
	mkdir -p $(OUT)
	mkarchiso -v -w /tmp/antenora-archiso -o $(OUT) $(PROFILE)

dante:
	CGO_ENABLED=0 go -C .. build -trimpath -ldflags '-s -w' \
		-o $(PROFILE)/airootfs/usr/bin/dante ./cmd/dante

clean:
	rm -rf /tmp/antenora-archiso $(OUT)
