# Gunkan Object Storage: Installation Manual

## Dependencies

The build process requires tools and libraries:
* [gRPC](https://grpc.io) and the [Go gRPC](https://github.com/grpc/grpc-go) implementation. 
* [Protobuf]() as part of [gRPC](https://grpc.io) 
* [Go >= 1.11](https://golang.org) with the support of [Go Modules](https://blog.golang.org/using-go-modules) enabled
* A set of modules described in [go.mod](./go.mod)

The deployment and the runtime requires additional tools:
* [Consul](https://consul.io)
* A decent implementation of TLS and its suite of tools, to generate certificates.

## Install from scratch

Build each of the set of ``gunkan`` commands.
```shell script
BASE=github.com/gunkan-io/object-storage
go mod download
go install ${BASE}/cmd/gunkan
go install ${BASE}/cmd/gunkan-data-gate
go install ${BASE}/cmd/gunkan-blob-store-fs
go install ${BASE}/cmd/gunkan-index-gate
go install ${BASE}/cmd/gunkan-index-store-rocksdb
```

## Deploy a sandbox

Gunkan provides [run.py](./ci/run.py), a tool to deploy test environments.

Use [run.py](./ci/run.py) to spawn a minimal environment deployed if subdirectories
of ``/tmp``, then hit ``Ctrl-C`` to make it exit gracefully):
```shell script
./ci/run.py
```