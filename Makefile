default: build/raspberry-pi-3/sync-putio

.PHONY: clean
clean:
	rm -rf build

.PHONY: all
all: build/raspberry-pi-3/sync-putio

build:
	mkdir build

build/raspberry-pi-3: build
	mkdir build/raspberry-pi-3

build/raspberry-pi-3/sync-putio: build/raspberry-pi-3
	env GOOS=linux GOARCH=arm GOARM=7 go build -o build/raspberry-pi-3/sync-putio
