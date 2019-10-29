//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "blob.hpp"

#include <unistd.h>
#include <fcntl.h>
#include <dirent.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/eventfd.h>

#include <gflags/gflags.h>
#include <glog/logging.h>
#include <libdill.h>

#include <cassert>
#include <algorithm>
#include <fstream>
#include <sstream>
#include <iostream>
#include <map>
#include <memory>
#include <string>
#include <utility>
#include <vector>


#include "strings.hpp"


#define URL_PREFIX_INFO "/info"
#define URL_PREFIX_V1_STATUS "/v1/status"
#define URL_PREFIX_V1_BLOB "/v1/blob"
#define URL_PREFIX_V1_LIST "/v1/list"

#define MESSAGE_INFO "gunkan object-storage blob v1"

#define FLAGS_OPEN O_WRONLY|O_CREAT|O_EXCL|O_CLOEXEC|O_NONBLOCK|O_NOATIME

static int _errno_to_http(int err) {
  switch (err) {
    case EINVAL:
      return 400;
    case ENOENT:
    case ENOTDIR:
      return 404;
    case EISDIR:
      return 502;
    case EBUSY:
      return 503;
    case EPERM:
    case EACCES:
    case EROFS:
      return 403;
    default:
      return 500;
  }
}

static size_t _index_next(const std::string &url, size_t start_offset);

static bool _mkdir(int fd_basedir, std::string path, bool retry) {
  errno = 0;
  if (0 == ::mkdirat(fd_basedir, path.c_str(), 0755))
    return true;
  if (!retry)
    return false;

  // Lazy creation of the parent
  auto parent = path_parent(path);
  if (!_mkdir(fd_basedir, parent, true))
    return false;

  errno = 0;
  return 0 == ::mkdirat(fd_basedir, path.c_str(), 0755);
}

size_t _index_next(const std::string &url, size_t i) {
  const size_t max = url.size();
  for (; i < max && url[i] == '/'; i++) {}
  return i;
}

static size_t bounded_length(const char *buf, size_t len) {
  size_t size{0};
  while (len-- && *buf++)
    size++;
  return size;
}

BlobServer::~BlobServer() {}

BlobServer::BlobServer() : stats{}, config{}, service(this) {}

bool BlobServer::configure() {
  return service.configure(config.basedir);
}

BlobService::~BlobService() {
  if (fd_basedir >= 0) {
    ::close(fd_basedir);
    fd_basedir = -1;
  }
}

BlobService::BlobService(BlobServer *srv) : server{srv}, fd_basedir{-1} {}

bool BlobService::configure(const std::string basedir) {
  if (fd_basedir >= 0) {
    SYSLOG(ERROR) << "Service already configured";
    return false;
  }

  fd_basedir = ::open(basedir.c_str(), O_RDONLY,
                      O_NOATIME | O_DIRECTORY | O_PATH);
  if (fd_basedir >= 0)
    return true;

  SYSLOG(ERROR) << "open(" << basedir << ") error: "
                << errno << '/' << ::strerror(errno);
  return false;
}

std::string BlobService::fullpath(const BlobId &id) {
  const unsigned int hd = server->config.hash_depth;
  const unsigned int hw = server->config.hash_width;
  std::stringstream ss;
  bool any{false};

  for (unsigned int i{0}; i < hd; ++i) {
    if (any)
      ss << '/';
    any = true;
    ss << id.id_content.substr(i * hw, hw);
  }
  if (any)
    ss << '/';
  ss << id.id_content.substr(hw * hd) << ',' << id.id_part << ','
     << id.position;
  return ss.str();
}

#define CASE(N) case N: server->stats.c_##N++; break

void BlobService::execute(
    std::unique_ptr<HttpRequest> req,
    std::unique_ptr<HttpReply> rep) {

  req->span_exec = req->tracer->StartSpan(
      "exec",
      {opentracing::FollowsFrom(&req->span_wait->context()),
       opentracing::ChildOf(&req->span_active->context())});

  if (_starts_with(req->url, URL_PREFIX_V1_BLOB)) {
    do_v1_blob(req.get(), rep.get());
  } else if (_starts_with(req->url, URL_PREFIX_V1_LIST)) {
    do_v1_list(req.get(), rep.get());
    server->stats.h_list++;
    server->stats.t_list += total_microseconds(*req, *rep);
  } else if (req->url == URL_PREFIX_V1_STATUS) {
    do_v1_status(req.get(), rep.get());
    server->stats.h_status++;
    server->stats.t_status += total_microseconds(*req, *rep);
  } else if (req->url == URL_PREFIX_INFO) {
    do_info(req.get(), rep.get());
    server->stats.h_info++;
    server->stats.t_info += total_microseconds(*req, *rep);
  } else {
    // No handler matched
    req->span_exec->Finish();
    req->span_active->Finish();
    rep->write_error(418);
    server->stats.h_other++;
    server->stats.t_other += total_microseconds(*req, *rep);
  }
  server->stats.b_in += req->bytes_in;
  server->stats.b_out += rep->bytes_out;

  switch (rep->code_) {
    CASE(200);
    CASE(201);
    CASE(204);
    CASE(206);
    CASE(400);
    CASE(403);
    CASE(404);
    CASE(405);
    CASE(408);
    CASE(409);
    CASE(418);
    CASE(499);
    CASE(502);
    CASE(503);
    default:
      server->stats.c_50X++;
      break;
  }
}

void BlobService::do_v1_blob(HttpRequest *req, HttpReply *rep) {
  // Check and normalize the chunkid
  BlobId id;
  do {
    std::string s = req->url.substr(
        _index_next(req->url, sizeof(URL_PREFIX_V1_BLOB) - 1));
    if (!id.decode(s)) {
      rep->write_error(400);
      return;
    }
  } while (0);
  const auto fp = server->service.fullpath(id);

  // Forward to the action depending on the method
  switch (req->GetMethod()) {
    case HttpMethod::Put:
      do_v1_blob_put(req, rep, fp);
      server->stats.h_put++;
      server->stats.t_put += total_microseconds(*req, *rep);
      return;
    case HttpMethod::Head:
      do_v1_blob_get(req, rep, fp, false);
      server->stats.h_head++;
      server->stats.t_head += total_microseconds(*req, *rep);
      return;
    case HttpMethod::Get:
      do_v1_blob_get(req, rep, fp, true);
      server->stats.h_get++;
      server->stats.t_get += total_microseconds(*req, *rep);
      return;
    case HttpMethod::Delete:
      do_v1_blob_delete(req, rep, fp);
      server->stats.h_delete++;
      server->stats.t_delete += total_microseconds(*req, *rep);
      return;
    default:
      rep->write_error(405);
      server->stats.h_other++;
      server->stats.t_other += total_microseconds(*req, *rep);
      return;
  }
}

void BlobService::do_v1_blob_delete(
    HttpRequest *req __attribute__((unused)),
    HttpReply *rep,
    const std::string &fp) {
  int rc = ::unlinkat(fd_basedir, fp.c_str(), 0);
  req->span_exec->Finish();
  req->span_active->Finish();
  return rep->write_error(!rc ? 204 : _errno_to_http(errno));
}

void BlobService::do_v1_blob_get(HttpRequest *req, HttpReply *rep,
    const std::string &path, bool body) {
  FD fd(::openat(fd_basedir, path.c_str(),
                 O_RDONLY | O_CLOEXEC | O_NOATIME | O_NONBLOCK));

  if (fd.fd < 0)
    return rep->write_error(_errno_to_http(errno));

  struct stat st{};
  int rc = ::fstat(fd.fd, &st);
  req->span_exec->Finish();
  req->span_active->Finish();
  if (0 > rc)
    return rep->write_error(_errno_to_http(errno));

  uint64_t size = st.st_size;
  rep->write_headers(body && size > 0 ? 200 : 204, size);
  if (!body || !size)
    return;

  while (size > 0) {
    ssize_t sent = rep->client->Sendfile(fd.fd, size);
    if (sent > 0) {
      size -= sent;
      rep->bytes_out += sent;
      ::dill_yield();
    } else if (sent < 0) {
      if (errno == EINTR)
        continue;
      if (errno == EAGAIN) {
        int64_t dl = ::dill_now() + 10000;
        if (::dill_fdout(rep->client->fd, dl) || ::dill_fdin(fd.fd, dl))
          return;
      } else {
        SYSLOG(WARNING)
            << "sendfile(" << path << ")"
            << " errno=" << errno << "/" << ::strerror(errno);
        return;
      }
    }
  }
}

void BlobService::do_v1_blob_put(
    HttpRequest *req,
    HttpReply *rep,
    const std::string &path) {
  FileAppender chunk;
  int err{0};
  bool retryable{true};

  std::string path_temp{path};
  path_temp += "@";

  // Open the temp file with a lazy autocreation of the directory
label_retry_open:
  chunk.fd = ::openat(fd_basedir, path_temp.c_str(), FLAGS_OPEN, 0644);
  if (chunk.fd < 0) {
    if (errno == ENOENT && retryable) {
      retryable = false;
      err = 0;
      if (_mkdir(fd_basedir, path_parent(path_temp), true))
        goto label_retry_open;
    }
    err = errno;
    goto label_reply;
  }

  // Check the existence of the final file
  if (0 == ::faccessat(fd_basedir, path.c_str(), F_OK, 0)) {
    LOG(INFO) << "Final chunk exists";
    err = EEXIST;
    goto label_rollback;
  }

  // Perform the upload with a special treatment for the body
  do {
    errno = 0;
    chunk.extend_allowed = server->config.flag_fallocate;
    auto rc = req->body->WriteTo(chunk);
    err = rc.first;
    req->bytes_in += rc.second;
  } while (0);

  if (err == 0) {
    // Validate the file under its final name & size
    if (0 != (err = chunk.truncate()))
      goto label_rollback;
    if (0 != ::renameat(fd_basedir, path_temp.c_str(),
                        fd_basedir, path.c_str())) {
      err = errno;
      goto label_rollback;
    }

    rep->write_error(201);

    // Post-Synchronous persistence
    // Advice the kernel cache about the content
    if (server->config.flag_fadvise_upload) {
      ::posix_fadvise64(chunk.fd, 0, chunk.written, POSIX_FADV_DONTNEED);
    }
    // Send a fsync() if configured so
    if (server->config.flag_fsync_data) {
      ::fdatasync(chunk.fd);
    } else if (server->config.flag_fsync_dir) {
      ::fdatasync(fd_basedir);
    }
  } else {
label_rollback:
    assert(err != 0);
    if (0 != ::unlinkat(fd_basedir, path_temp.c_str(), 0)) {
      if (errno != ENOENT)
        SYSLOG(ERROR) << "Orphan temporary chunk: " << path_temp;
    }
label_reply:
    assert(err != 0);
    req->span_exec->Finish();
    req->span_active->Finish();
    return rep->write_error(_errno_to_http(err));
  }
}

class Walker {
 private:
  std::stringstream &ss_;
  unsigned int w_;
  std::string marker_;

 public:
  ~Walker() {}

  Walker(std::stringstream &ss, unsigned int w):
      ss_{ss}, w_{w}, marker_{} {}

  void set_marker(std::string m) { marker_.assign(m); }

  int run(unsigned int max, std::string path, std::string prefix) {
    std::vector<std::string> dirs, files;
    unsigned int total{0};

    DIR *d = opendir(path.c_str());
    assert(d != nullptr);

    for (;;) {
      struct dirent *out = ::readdir(d);
      if (!out)
        break;
      if (out->d_name[0] == '.')
        continue;
      const size_t length = bounded_length(out->d_name, w_ + 1);
      switch (out->d_type) {
        case DT_DIR:
          if (w_ == length)
            dirs.emplace_back(out->d_name);
          break;
        case DT_REG:
          if (w_ + 1 == length) {
            std::stringstream ss;
            ss << prefix << out->d_name;
            std::string s = ss.str();
            BlobId blobid;
            if (blobid.decode(s))
              files.emplace_back(std::move(s));
          }
          break;
      }
    }
    closedir(d);
    d = nullptr;

    if (!files.empty()) {
      // TODO(jfs): should we filter before sorting?
      std::sort(files.begin(), files.end());
      for (auto &f : files) {
        if (marker_.empty() || f > marker_) {
          ss_ << f << CRLF;
          total++;
        }
      }
    }

    // Recurse on hashed directories
    if (!dirs.empty()) {
      std::sort(dirs.begin(), dirs.end());
      for (const auto &d : dirs) {
        if (total >= max)
          break;
        if (marker_.empty() || d > marker_ || _starts_with(marker_, d))
          total += run(max - total, path + '/' + d, prefix + d);
      }
    }

    return total;
  }
};

void BlobService::do_v1_list(HttpRequest *req, HttpReply *rep) {
  auto marker = req->url.substr(
      _index_next(req->url, sizeof(URL_PREFIX_V1_LIST)-1));

  std::stringstream ss;
  Walker walker(ss, server->config.hash_width);
  if (!marker.empty())
    walker.set_marker(marker);
  walker.run(1000, server->config.basedir, "");
  std::string body = ss.str();

  rep->headers["Content-Type"] = "text/plain";
  req->span_exec->Finish();
  req->span_active->Finish();
  rep->write_headers(200, -1);
  rep->write_chunk(body.data(), body.size());
  rep->write_final_chunk();
}


#define FIELD(N) StatField {offsetof(BlobStats, N), #N}

struct StatField { uint32_t offt; const char *name; };

static StatField all_fields[] = {
  FIELD(b_in),
  FIELD(b_out),

  FIELD(t_info),
  FIELD(t_status),
  FIELD(t_put),
  FIELD(t_get),
  FIELD(t_head),
  FIELD(t_delete),
  FIELD(t_list),
  FIELD(t_other),

  FIELD(h_info),
  FIELD(h_status),
  FIELD(h_put),
  FIELD(h_get),
  FIELD(h_head),
  FIELD(h_delete),
  FIELD(h_list),
  FIELD(h_other),

  FIELD(c_200),
  FIELD(c_201),
  FIELD(c_204),
  FIELD(c_206),
  FIELD(c_400),
  FIELD(c_403),
  FIELD(c_404),
  FIELD(c_405),
  FIELD(c_408),
  FIELD(c_409),
  FIELD(c_418),
  FIELD(c_499),
  FIELD(c_502),
  FIELD(c_503),
  FIELD(c_50X),
};

static std::string _encode_json(const BlobStats &st) {
  auto base = reinterpret_cast<const uint8_t*>(&st);
  std::stringstream builder;
  char separator = '{';
  for (const auto &f : all_fields) {
    const uint64_t v = *reinterpret_cast<const uint64_t*>(base + f.offt);
    builder << separator << '"' << f.name << '"' << ':' << v;
    separator = ',';
  }
  builder << '}';
  return builder.str();
}

void BlobService::do_v1_status(HttpRequest *req, HttpReply *rep) {
  rep->headers["Content-Type"] = "text/json";
  std::string s = _encode_json(server->stats);
  req->span_exec->Finish();
  req->span_active->Finish();
  rep->write_headers(200, -1);
  rep->write_chunk(s.data(), s.size());
  rep->write_final_chunk();
}

void BlobService::do_info(HttpRequest *req, HttpReply *rep) {
  rep->headers["Content-Type"] = "text/plain";
  req->span_exec->Finish();
  req->span_active->Finish();
  rep->write_headers(200, -1);
  rep->write_chunk(const_cast<char *>(MESSAGE_INFO), sizeof(MESSAGE_INFO) - 1);
  rep->write_final_chunk();
}
