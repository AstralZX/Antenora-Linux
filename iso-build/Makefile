# Build the Antenora ISO from source. No Arch, no mkarchiso.
# Usage: make iso VERSION=1.0.0

VERSION ?= 1.0.0
ROOT    := ..

.PHONY: all iso stage3 clean

all: iso

iso:
	$(ROOT)/scripts/iso-build.sh $(VERSION)

stage3:
	$(ROOT)/scripts/stage3-build.sh $(VERSION)

clean:
	rm -rf $(ROOT)/dist/iso $(ROOT)/dist/iso-work $(ROOT)/dist/sysroot
