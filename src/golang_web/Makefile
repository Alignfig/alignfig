.PHONY: build clean

versionflags?=
module?=github.com/alex123012/alignfig/src/golang_web
binary?=bin/golang_web
extldflags?=
ldflags?=
_flags=-v -a -tags netgo -ldflags="-extldflags '-static $(extldflags)' -s -w $(versionflags) $(ldflags)"

build:
	CGO_ENABLED=0 go build $(_flags) -o $(binary) $(module)
	chmod +x $(binary)
clean:
	rm -f $$GOPATH/$(binary)
	rm -f bin/*
