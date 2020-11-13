generate: mockgen
	go generate ./...

mockgen:
	GO111MODULE=on go get -v github.com/golang/mock/mockgen@latest

test: generate
	go test -v ./... -count=1
