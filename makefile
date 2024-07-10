BUILD_DIR=./build

build:
	CGO_ENABLED=0 GOOS=linux  GOARCH=amd64  go build -o build/friends main.go
