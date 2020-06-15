BIN := bin
APP := ymlclient
OUT := $(BIN)/$(APP)

clean:
	-@rm $(OUT)

build:
	go build -o $(OUT) cmd/console/main.go
.PHONY: build

run: build
	$(OUT)
.PHONY: run

check:
	golangci-lint run
