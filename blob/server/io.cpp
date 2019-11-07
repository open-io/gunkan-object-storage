//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "io.hpp"

#include <unistd.h>
#include <fcntl.h>
#include <sys/uio.h>
#include <sys/sendfile.h>

#include <glog/logging.h>
#include <libdill.h>

#include <cassert>
#include <cstdlib>
#include <cstdint>
#include <cerrno>

ssize_t _read_at_least(int fd, uint8_t *base, size_t max, size_t min,
    int64_t dl) {
  size_t total{0};
  while (total < min) {
    ssize_t r = ::read(fd, base + total, max - total);
    if (r < 0) {
      if (errno == EINTR)
        continue;
      if (errno == EAGAIN && !::dill_fdin(fd, dl))
        continue;
      return -1;
    }
    if (r == 0)
      return -1;
    total += r;
  }
  return total;
}

bool _write_full(int fd, const uint8_t *buf, size_t len, int64_t dl) {
  size_t total{0};
  while (total < len) {
    ssize_t w = ::write(fd, buf + total, len - total);
    if (w < 0) {
      if (errno == EINTR)
        continue;
      if (errno == EAGAIN && 0 == ::dill_fdout(fd, dl))
        continue;
      return false;
    }
    total += w;
  }
  return true;
}

bool _writev_full(int fd, struct iovec *iov, size_t nb, int64_t dl) {
  size_t total{0};

  // Compute the total length of the iov
  for (size_t i{0}; i < nb; ++i)
    total += iov[i].iov_len;

  while (total > 0) {
    ssize_t w = ::writev(fd, iov, nb);
    if (w < 0) {
      if (errno == EINTR)
        continue;
      if (errno == EAGAIN && 0 == ::dill_fdout(fd, dl))
        continue;
      return false;
    }
    while (w > 0) {
      const size_t l0 = iov[0].iov_len;
      if (l0 > static_cast<size_t>(w)) {
        iov[0].iov_base = static_cast<uint8_t *>(iov[0].iov_base) + w;
        iov[0].iov_len -= w;
        total -= w;
        w = 0;
      } else {
        w -= l0;
        total -= l0;
        ++iov;
        --nb;
      }
    }
    total -= w;
  }
  return true;
}

bool _writev_full(int fd, std::vector<iovec> &iov, int64_t dl) {
  return _writev_full(fd, iov.data(), iov.size(), dl);
}

bool _writev_full(int fd, std::vector<Slice> &slices, int64_t dl) {
  std::vector<iovec> iov;
  for (auto &s : slices)
    iov.push_back({s.data(), s.size()});
  return _writev_full(fd, iov, dl);
}

size_t compute_iov_size(iovec *iov, size_t nb) {
  size_t total{0};
  if (iov) {
    for (size_t i{0}; i < nb; ++i)
      total += iov[i].iov_len;
  }
  return total;
}

FD::FD() : fd{-1} {}

FD::FD(int f) : fd{f} {}

FD::~FD() { close(); }

bool FD::valid() const { return fd >= 0; }

void FD::detach() {
  if (fd >= 0) {
    ::dill_fdclean(fd);
  }
}

void FD::close() {
  detach();
  if (fd >= 0) {
    ::close(fd);
    fd = -1;
  }
}

ActiveFD::ActiveFD(int f, const NetAddr &a): FD(f), peer{a} {}

ActiveFD::~ActiveFD() {}

void ActiveFD::SetPrio(IpTos tos) {
  int opt = static_cast<int>(tos);
  setsockopt(fd, SOL_SOCKET, SO_PRIORITY, &opt, sizeof(opt));
}

int ActiveFD::Sendfile(int from, int64_t size) {
  return ::sendfile(fd, from, nullptr, size);
}

int ActiveFD::ReadIn(std::shared_ptr<Block> block, Slice *out, int64_t dl) {
  ssize_t r;

label_retry:
  r = ::recv(fd, block->data(), block->size(), MSG_NOSIGNAL);
  if (r < 0) {
    if (errno == EINTR)
      goto label_retry;
    if (errno == EAGAIN) {
      if (!::dill_fdin(fd, dl))
        goto label_retry;
      return ETIMEDOUT;
    } else {
      return errno;
    }
  }

  *out = Slice(std::move(block), 0, r);
  return 0;
}

int ActiveFD::Read(Slice *out, int64_t dl) {
  std::shared_ptr<Block> block(new ArrayBlock<2048>());
  return ReadIn(block, out, dl);
}

FileAppender::FileAppender() : FD(), written{0}, allocated{0} {}

FileAppender::FileAppender(int f) : FD(f), written{0}, allocated{0} {}

FileAppender::~FileAppender() {}

void FileAppender::preallocate(int64_t size) {
  if (!extend_allowed)
    return;
  if (0 == ::fallocate(fd, FALLOC_FL_KEEP_SIZE, written, size)) {
    allocated += size;
  } else {
    if (errno == ENOTSUP)
      extend_allowed = false;
  }
}

int FileAppender::truncate() {
  if (written > allocated)
    return ::ftruncate(fd, written) ? errno : 0;
  return 0;
}

int FileAppender::sendfile(int from, int64_t size) {
  int64_t sent{0};
  while (sent < size) {
    ssize_t sz = ::sendfile(fd, from, nullptr, size - sent);
    switch (sz) {
      default:
        sent += sz;
        // FALLTHROUGH
      case 0:
        continue;
      case -1:
        switch (errno) {
          case EAGAIN:
            ::dill_fdin(from, ::dill_now() + 1000);
            // FALLTHROUGH
          case EINTR:
            continue;
          default:
            return -1;
        }
    }
  }
  return 0;
}


static int _dump(int fd, uint64_t &written, uint64_t &accumulated, Pipe &p) {
  for (;;) {
    ssize_t nw = p.spliceTo(fd, accumulated);
  ::dill_yield();
    switch (nw) {
      case -1:
        switch errno {
          case EAGAIN:
            ::dill_fdout(fd, ::dill_now() + 1000);
            // FALLTHROUGH
          case EINTR:
            continue;
          default:
            return errno;
        }
      case 0:
        return EOF;
      default:
        written += nw;
        accumulated -= nw;
        return 0;
    }
  }
}

int FileAppender::splice(int from, int64_t size) {
  const auto usize = static_cast<uint64_t>(size);
  const int64_t extent{64 * 1024 * 1024};
  size_t batch{8 * 1024 * 1024};
  Pipe p;

  // Get a large pipe
  if (!p.init()) {
    return errno;
  } else {
    int arg = batch;
    ::fcntl(p.head(), F_SETPIPE_SZ, arg);
    batch = ::fcntl(p.head(), F_GETPIPE_SZ);
  }

  uint64_t accumulated{0};
  int err{0};
  bool full{false};

  while (!err && (size < 0 || accumulated + written < usize)) {
    // Load the pipe but not too much
    for (int i{0}; i++ < 4 && accumulated < batch;) {
      ssize_t nr = p.spliceFrom(from, batch);
      switch (nr) {
        case -1:
          switch (errno) {
            case EAGAIN:
              if (accumulated > 0)
                goto label_dump;
              ::dill_fdin(from, ::dill_now() + 5000);
              // FALLTHROUGH
            case EINTR:
              continue;
            default:
              return errno;
          }
        case 0:
          goto label_finish;
        default:
          accumulated += nr;
          continue;
      }
    }

label_dump:
    if (written + accumulated < allocated)
      preallocate(extent);
    if (accumulated > 0)
      err = _dump(fd, written, accumulated, p);
  }

label_finish:
  if (!err && accumulated > 0)
    err = _dump(fd, written, accumulated, p);

  assert(accumulated == 0 || err != 0);

  return err;
}

Pipe::Pipe() : fd{-1, -1} {}

Pipe::~Pipe() {
  if (fd[0] >= 0)
    ::close(fd[0]);
  if (fd[1] >= 0)
    ::close(fd[1]);
  fd[0] = fd[1] = -1;
}

bool Pipe::init() { return 0 == ::pipe2(fd, O_NONBLOCK); }

int Pipe::head() const { return fd[0]; }

int Pipe::tail() const { return fd[1]; }

ssize_t Pipe::spliceTo(int dst, size_t n) const {
  return ::splice(head(), nullptr, dst, nullptr, n,
                  SPLICE_F_NONBLOCK|SPLICE_F_MOVE|SPLICE_F_GIFT);
}

ssize_t Pipe::spliceFrom(int src, size_t n) const {
  return ::splice(src, nullptr, tail(), nullptr, n,
                  SPLICE_F_NONBLOCK|SPLICE_F_MOVE|SPLICE_F_GIFT);
}
