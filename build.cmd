@REM https://habr.com/ru/post/249449/

@SET GOOS=windows
@SET GOARCH=amd64
go build -ldflags "-s -w" -o bin/cacheserver_amd64.exe

@SET GOOS=linux
@SET GOARCH=386
go build -ldflags "-s -w" -o bin/cacheserver_i386

@SET GOOS=linux
@SET GOARCH=amd64
go build -ldflags "-s -w" -o bin/cacheserver_amd64

@SET GOOS=linux
@SET GOARCH=arm
@SET GOARM=7
go build -ldflags "-s -w" -o bin/cacheserver_armv7

@SET GOOS=linux
@SET GOARCH=arm64
go build -ldflags "-s -w" -o bin/cacheserver_aarch64

@SET GOOS=darwin
@SET GOARCH=amd64
go build -ldflags "-s -w" -o bin/cacheserver_darwin