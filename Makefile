BASE=github.com/jfsmig/object-storage
GO=go
PROTOC=protoc

AUTO=
AUTO+= pkg/blobindex-proto/blobindex.pb.go

all: prepare
	$(GO) install $(BASE)/cmd/gunkan
	$(GO) install $(BASE)/cmd/gunkan-blob
	$(GO) install $(BASE)/cmd/gunkan-blobindex-rocksdb

prepare: $(AUTO)

pkg/blobindex-proto/%.pb.go: api/blobindex.proto
	$(PROTOC) -I api api/blobindex.proto --go_out=plugins=grpc:pkg/blobindex-proto

clean:
	-rm $(AUTO)

.PHONY: all prepare clean test bench fmt

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

