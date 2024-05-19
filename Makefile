all:
	go build --buildmode=plugin ./plugins/kallisti
	go build


windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build --buildmode=exe
