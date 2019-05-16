build:
	go build -o bin/webhook .

build-linux64:
	env GOOS=linux GOARCH=amd64 go build -o bin/webhook .
