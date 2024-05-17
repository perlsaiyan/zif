# Zif, the quick little MUD client

Just messing around with golang, TCP connections, MSDP parsing, bubbletea, etc.

Not sure what the plan is for this code, pushing to github in case it helps
someone else and so I don't lose it.


## To cross compile for windows
```bash
 pacman -S mingw-w64-gcc
 GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build --buildmode=exe
```


