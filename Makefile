BASE=github.com/jfsmig/object-storage
GO=go
PROTOC=protoc

AUTO=
AUTO+= pkg/gunkan-index-proto/index.pb.go

all: prepare
	$(GO) install $(BASE)/cmd/gunkan
	$(GO) install $(BASE)/cmd/gunkan-data-gate
	$(GO) install $(BASE)/cmd/gunkan-blob-store-fs
	$(GO) install $(BASE)/cmd/gunkan-index-gate
	$(GO) install $(BASE)/cmd/gunkan-index-store-rocksdb

prepare: $(AUTO)

pkg/gunkan-index-proto/%.pb.go: api/index.proto
	$(PROTOC) -I api api/index.proto --go_out=plugins=grpc:pkg/gunkan-index-proto

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

