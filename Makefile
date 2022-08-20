build:
	go build

test:
	go test -tags=unit -v -coverprofile=coverprofile.out -covermode=count ./...
