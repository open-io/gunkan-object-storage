# Gunkan - Object Storage - Blob service

**gunkan-blob** serves BLOB's hosted on a local filesystem.

The protocol is a subset of HTTP/1.1 over TCP/IP connections:
* `HTTP/1.1` is the only protocol version accepted
* `Connection: keepalive` is ignored, the connections are always closed and `Connection: close` is invariably returned
* `Expect: 100-continue` is honored
* `Transfer-Encoding: chunked` and `Transfer-Encoding: inline` are honored. `inline` is implied when nothing is mentioned.
* Redirections are never emitted.


## Server

Written in C++ with the subsequent libraries:
* `pthread`
* `libdill` to run one pool of coroutine into each worker thread.
* `http-parser`
* `glog`
* `opentracing`

Each service uses 4 threads by default:
* The main thread emits load-balancing tokens.
  * It manages no coroutine and does not require `libdill`
  * The tokens are collected through event file descriptors, one par thread.
* A *classifier* thread is responsible of
  * accepting connections
  * instanciating HTTP request objects
  * parse the header
  * Authenticate the emitter 
  * pass the request to an executor thread
* 2 threads *Best-Effort* manage the execution of requests
  * one thread is for PUT/DELETE
  * one thread is for GET/HEAD.

If the program is started as root:
* 2 additonal threads *Real-Time* also manage execution of requests:
  * one for PUT/DELETE and one for GET/HEAD.
  * Both have a higher CPU priority than the *Best-Effort* threads
  * Their relative CPUpriority can be tuned.
* The relative CPU priority of both *Best-Effort* threads can be adpated.
* The CPU priority of the *classifier* thread can also be adapted.


## Build, Install, Run

```
cmake
make
make install
```


## API

Special request:
* ``BLOB-ID`` is a unique identifier for a BLOB.
  The format is a hexadecimal string of 64 characters.

Common Request headers:
* `Host` must be present and valued to the identifier of the service as known
  in the Catalog.
* `X-gunkan-token` must be present and valued as an authentication / authorization
  token as issued by the Access Manager

### GET /info

Returns a description of the service.

### PUT /v1/blob/{BLOB-ID}

Add a BLOB on the storage of the service.

### DELETE /v1/blob/{BLOB-ID}

Remove a BLOB from the storage of the service.

### GET /v1/blob/{BLOB-ID}

Fetch a BLOB. The data will be served as the body and the metadata will be
present in the header fields of the reply.

### HEAD /v1/blob/{BLOB-ID}

Fetch metadata information about a BLOB. The metadata will be present as fields
in the header of the reply.

That route respects the semantics of a HEAD HTTP request: e.g. the `content-length`
field is present but no body is expected.

### GET /v1/status


