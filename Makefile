get:
	go get github.com/chuckpreslar/inflect
	go get github.com/serenize/snaker

build:
	go build -o main main.go

install: get
	go install github.com/mcos/schemabuf