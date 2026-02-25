.PHONY: all clean

VERSION = $(shell go run cmd/mdify/main.go -v | cut -c15-)
BINARY_NAME = mdify
SOURCE = cmd/mdify/main.go

# Function to build for a specific platform
# Usage: $(call build,platform_dir,GOOS,GOARCH,binary_name)
define build
	mkdir -p dist/$(1)
	env GOOS=$(2) GOARCH=$(3) go build -o dist/$(1)/$(4) $(SOURCE)
endef

# Function to create zip archive
# Usage: $(call zip_archive,platform_dir,binary_name,archive_suffix)
define zip_archive
	cd dist/$(1) && zip $(BINARY_NAME)-$(VERSION)_$(3).zip $(2) && mv $(BINARY_NAME)-$(VERSION)_$(3).zip ../
endef

all: release

test:
	go test -v ./...

windows_64:
	$(call build,windows_64,windows,amd64,mdify.exe)

mac_intel:
	$(call build,mac_intel,darwin,amd64,mdify)

mac_silicon:
	$(call build,mac_silicon,darwin,arm64,mdify)

linux_64:
	$(call build,linux_64,linux,amd64,mdify)

release: windows_64 mac_intel mac_silicon linux_64
	$(call zip_archive,windows_64,mdify.exe,windows64)
	$(call zip_archive,mac_intel,mdify,mac_intel)
	$(call zip_archive,mac_silicon,mdify,mac_silicon)
	$(call zip_archive,linux_64,mdify,linux64)
	cd dist/linux_64 && tar -czvf $(BINARY_NAME)-$(VERSION)_linux64.tar.gz mdify && mv $(BINARY_NAME)-$(VERSION)_linux64.tar.gz ../

clean:
	rm -rf dist/

