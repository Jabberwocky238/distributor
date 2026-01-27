.PHONY: build run clean

build:
	mkdir -p bin
	go build -o bin/distributor .

dev:
	go run main.go

clean:
	rm -rf bin
