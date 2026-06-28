.PHONY: build run clean

BINARY := paddleocrvl.exe

build:
	go build -o $(BINARY) ./cmd/ocr/main.go

run:
	go run ./cmd/ocr/main.go

clean:
	rm -f $(BINARY)
