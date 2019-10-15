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

#include <libdill.h>

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

void ActiveFD::set_priority(IpTos tos) {
  int opt = static_cast<int>(tos);
  setsockopt(fd, SOL_SOCKET, SO_PRIORITY, &opt, sizeof(opt));
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

int FileAppender::splice(int from, int64_t size) {
  const int64_t extent{64 * 1024 * 1024}, batch{8 * 1024 * 1024};
  Pipe p;
  bool eof{false};

  if (!p.init())
    return errno;

  while (!eof && written < size) {
    int64_t accumulated{0};

    ::dill_yield();

    // Load the pipe
    while (accumulated < batch && !eof) {
      ssize_t rc = ::splice(
          from, nullptr, p.head(), nullptr,
          batch, SPLICE_F_NONBLOCK | SPLICE_F_MOVE);
      switch (rc) {
        case -1:
          if (errno == EINTR)
            continue;
          if (errno == EAGAIN) {
            if (accumulated > 0)
              goto label_dump;
            ::dill_fdin(from, ::dill_now() + 2000);
            continue;
          }
          return errno;
        case 0:
          eof = true;
          continue;
        default:
          accumulated += rc;
          continue;
      }
    }

label_dump:
    // Maybe extend the output
    if (written + accumulated < allocated) {
      preallocate(extent);
    }

    // Dump the pipe
    while (accumulated > 0) {
      ssize_t rc = ::splice(p.tail(), nullptr, fd, nullptr,
                            static_cast<size_t>(accumulated),
                            SPLICE_F_NONBLOCK | SPLICE_F_MOVE);
      switch (rc) {
        case -1:
          if (errno == EINTR)
            continue;
          if (errno == EAGAIN) {
            ::dill_fdout(fd, ::dill_now() + 2000);
            continue;
          }
          return errno;
        case 0:
          return EBADF;
        default:
          written += rc;
          accumulated -= rc;
          continue;
      }
    }
  }

  return 0;
}

Pipe::Pipe() : fd{-1, -1} {}

Pipe::~Pipe() {
  if (fd[0] >= 0)
    ::close(fd[0]);
  if (fd[1] >= 0)
    ::close(fd[1]);
  fd[0] = fd[1] = -1;
}

bool Pipe::init() { return 0 == ::pipe(fd); }

int Pipe::head() const { return fd[0]; }

int Pipe::tail() const { return fd[1]; }
