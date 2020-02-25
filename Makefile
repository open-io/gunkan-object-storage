BASE=github.com/jfsmig/object-storage
GO=go
PROTOC=protoc

AUTO=
AUTO+= pkg/kv-proto/kv.pb.go

all: prepare
	$(GO) install $(BASE)

prepare: $(AUTO)

pkg/kv-proto/%.pb.go: api/kv.proto
	$(PROTOC) -I api api/kv.proto --go_out=plugins=grpc:pkg/kv-proto

clean:
	-rm $(AUTO)

.PHONY: all prepare clean test bench fmt try

fmt:
	find * -type f -name '*.go' \
		| grep -v -e '_auto.go$$' -e '.pb.go$$' \
		| while read F ; do dirname $$F ; done \
		| sort | uniq | while read D ; do ( set -x ; cd $$D && go fmt ) done

test: all
	find * -type f -name '*_test.go' \
		| while read F ; do dirname $$F ; done \
		| sort | uniq | while read D ; do ( set -x ; cd $$D && go test ) done

bench: all
	find * -type f -name '*_test.go' \
		| while read F ; do dirname $$F ; done \
		| sort | uniq | while read D ; do ( set -x ; cd $$D && go -bench=. test ) done

