//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "internals.hpp"

BlobServer::~BlobServer() {}

BlobServer::BlobServer() : stats{}, config{}, threads(this), service(this) {}

bool BlobServer::running() const { return threads.running(); }

void BlobServer::configure(int fd) {
  threads.worker_ingress.fd_server = fd;
}

BlobService::~BlobService() {}

BlobService::BlobService(BlobServer *srv) : server{srv} {}

std::string BlobService::fullpath(const BlobId &id) {
  const unsigned int hd = server->config.hash_depth;
  const unsigned int hw = server->config.hash_width;
  std::stringstream ss;
  ss << server->config.basedir;
  for (unsigned int i{0}; i < hd; ++i) {
    ss << '/' << id.id_content.substr(i * hw, hw);
  }
  ss << '/' << id.id_content.substr(hw * hd) << ',' << id.id_part << ',' << id.position;
  return ss.str();
}

void BlobService::execute(HttpRequest *req, HttpReply *rep) {
  if (_starts_with(req->url, URL_PREFIX_V1_BLOB)) {
    (void) do_v1_blob(req, rep);
  } else if (req->url == URL_PREFIX_V1_STATUS) {
    (void) do_v1_status(req, rep);
  } else if (req->url == URL_PREFIX_INFO) {
    (void) do_info(req, rep);
  } else {
    // No handler matched
    (void) rep->write_error(404);
  }

  // No connection kept alive
  ::dill_fdclean(rep->client->fd);
  ::close(rep->client->fd);
}

bool BlobService::do_v1_blob(HttpRequest *req, HttpReply *rep) {
  BlobId id;

  // Check and normalize the chunkid
  auto chunkid = req->url.substr(sizeof(URL_PREFIX_V1_BLOB)-1);
  if (!id.decode(chunkid))
    return rep->write_error(400);

  auto fp = req->client->server->service.fullpath(id);
  DLOG(INFO) << "chunkid = " << chunkid;
  DLOG(INFO) << "path = " << fp;
  DLOG(INFO) << "body = " << req->body.size() - req->body_offset;

  // Forward to the action depending on the method
  switch (req->parser.method) {
    case HTTP_PUT:
      return do_v1_blob_put(req, rep);
    case HTTP_HEAD:
      return do_v1_blob_get(req, rep, false);
    case HTTP_GET:
      return do_v1_blob_get(req, rep, true);
    case HTTP_DELETE:
      return do_v1_blob_delete(req, rep, fp);
    default:
      return rep->write_error(405);
  }
}

bool BlobService::do_v1_blob_delete(HttpRequest *req, HttpReply *rep,
    const std::string &fp) {
  LOG(INFO) << "DELETE " << fp;
  (void) req;
  return rep->write_error(501);
}

bool BlobService::do_v1_blob_get(HttpRequest *req, HttpReply *rep, bool body) {
  (void) req, (void) body;
  return rep->write_error(501);
}

bool BlobService::do_v1_blob_put_chunked(HttpRequest *req, HttpReply *rep, int fd_chunk) {
  (void) req, (void) fd_chunk;
  return rep->write_error(501);
}

bool BlobService::do_v1_blob_put_inline(HttpRequest *req, HttpReply *rep, int fd_chunk) {
  (void) req, (void) fd_chunk;
  return rep->write_error(501);
}

bool BlobService::do_v1_blob_put(HttpRequest *req, HttpReply *rep) {
  std::string path;
  int fd_chunk{-1};
  volatile bool retryable{false};
  const int flags_open{O_WRONLY|O_CREAT|O_EXCL|O_CLOEXEC|O_NONBLOCK};

  // Open the target file with a lazy autocreation of the directory
label_retry_open:
  fd_chunk = ::open(path.c_str(), flags_open, 0644);
  if (fd_chunk < 0) {
    switch (errno) {
      case EPERM:
        return rep->write_error(403);
      case ENOENT:
        if (retryable) {
          retryable = false;
          goto label_retry_open;
        }
        return rep->write_error(403);
      case ENOTDIR:
        return rep->write_error(500);
    }
  }

  // TODO(jfs): Pre-allocate the file
#ifdef __linux
  do {
    size_t _length{NS_CHUNK_SIZE};
    if (req->parser.content_length > 0) {
      if (!(req->parser.flags & F_CHUNKED)) {
        _length = req->parser.content_length;
      }
    }
    ::fallocate(fd_chunk, FALLOC_FL_KEEP_SIZE, 0, _length);
  } while (0);
#endif

  // TODO(jfs): give the kernel an advice on further accesses

  // Perform the upload with a special treatment for the body
  bool done{false};
  if (req->parser.flags & F_CHUNKED) {
    done = do_v1_blob_put_chunked(req, rep, fd_chunk);
  } else {
    done = do_v1_blob_put_inline(req, rep, fd_chunk);
  }

  if (done) {
    // TODO(jfs): relink the chunk under its final name
    // TODO(jfs): send the fsync if configured so
  } else {
    ::unlink(path.c_str());
  }

  ::close(fd_chunk);
  return done;
}

bool BlobService::do_v1_status(HttpRequest *req, HttpReply *rep) {
  (void) req;
  std::stringstream ss;
  ss << "plop" << "\r\n";
  auto s = ss.str();
  rep->write_headers(200, -1);
  rep->write_chunk(s.data(), s.size());
  rep->write_final_chunk();
  return true;
}

#define MESSAGE_INFO "gunkan object-storage blob v1"

bool BlobService::do_info(HttpRequest *req, HttpReply *rep) {
  (void) req;
  return rep->write_headers(200, -1)
    && rep->write_chunk(const_cast<char*>(MESSAGE_INFO), sizeof(MESSAGE_INFO)-1)
    && rep->write_final_chunk();
}
