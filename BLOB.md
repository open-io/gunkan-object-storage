# Gunkan - Object Storage - Blob service

**gunkan-blob** serves BLOB's hosted on a local filesystem.

The protocol is a subset of HTTP/1.1 over TCP/IP connections:
* `HTTP/1.1` is the only protocol version accepted
* `Connection: keepalive` is ignored, the connections are always closed and `Connection: close` is invariably returned
* `Expect: 100-continue` is honored
* `Transfer-Encoding: chunked` and `Transfer-Encoding: inline` are honored. `inline` is implied when nothing is mentioned.
* Redirections are never emitted.


## Build, Install, Run

```
cmake
make
make install
```


## API

Special request fields:
* ``BLOB-ID`` is a unique identifier for a BLOB.
  The format is a hexadecimal string of 64 characters.

Common Request headers:
* `Host` must be present and valued to the identifier of the service as known
  in the Catalog.
* `X-gunkan-token` must be present and valued as an authentication / authorization
  token as issued by the Access Manager

### GET /info

Returns a description of the service.

### GET /v1/status

Returns usage statistics about the current service.
The body contains JSON

### GET /v1/list

Returns a list of ``{BLOB-ID}``, one per line, with en `CRLF` as a line separator.
 
Optional query string arguments are honored:
* ``marker`` a prefix of a ``{BLOB-ID}`` that must be past by the iterator.
* ``max`` the maximum number of items in the answer

### PUT /v1/blob/{BLOB-ID}

Add a BLOB on the storage of the service.

### GET /v1/blob/{BLOB-ID}

Fetch a BLOB. The data will be served as the body and the metadata will be
present in the header fields of the reply.

### HEAD /v1/blob/{BLOB-ID}

Fetch metadata information about a BLOB. The metadata will be present as fields
in the header of the reply.

That route respects the semantics of a HEAD HTTP request: e.g. the `content-length`
field is present but no body is expected.

### DELETE /v1/blob/{BLOB-ID}

Remove a BLOB from the storage of the service.
