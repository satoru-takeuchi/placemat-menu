# Run github.com/rakyll/statik by `go generate`

SOURCES := $(wildcard cmd/placemat-menu/public/*/*)
TARGET := cmd/placemat-menu/statik/statik.go

all: $(TARGET)
	go install ./...

$(TARGET): $(SOURCES)
	mkdir -p $(dir $(TARGET))
	go get github.com/rakyll/statik/...
	go generate ./...

.PHONY:	all
