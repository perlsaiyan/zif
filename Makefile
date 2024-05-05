all:
	go build --buildmode=plugin ./plugins/kallisti/kallisti.go
	go build
