.PHONY: build run clean

BINARY := paddleocrvl.exe

build:
	go build -o $(BINARY) ./cmd/main.go

run:
	go run ./cmd/main.go

clean:
	rm -f $(BINARY)
