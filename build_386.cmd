set GOARCH=386
set CGO_ENABLED=1
set CC=C:\TDM-GCC-64\bin\gcc
set CGO_CFLAGS=-g -O2 -m32
set CGO_LDFLAGS=-g -O2 -w -s
go build -x -o bin/x86/indxr.exe ./cmd/indexer
upx bin/x86/indxr.exe