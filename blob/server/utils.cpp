//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "internals.hpp"

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
        iov[0].iov_base = static_cast<uint8_t*>(iov[0].iov_base) + w;
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

bool _starts_with(const std::string &s, const std::string &p) {
  return s.length() >= p.length() && !memcmp(s.data(), p.data(), p.length());
}

int64_t microseconds() {
  struct timespec ts{};
  (void) clock_gettime(CLOCK_MONOTONIC, &ts);
  return (ts.tv_nsec / 1000) + (ts.tv_sec * 1000000 );
}

/**
 * String validator based on a set of accepted characters.
 * Builds a bitmap upon initiation.
 */
class CharSet {
 private:
  std::array<bool, 256> allowed;

 public:
  ~CharSet() {}
  CharSet() : allowed{} {}
  explicit CharSet(const char *s) : CharSet() {
    for (; *s; ++s)
      allowed[*s] = true;
  }
  bool validate(const std::string &s) {
    for (auto c : s) {
      if (!allowed[c]) return false;
    }
    return true;
  }
};

static CharSet _hexa("0123456789ABCDEFabcdef");

bool is_hexa(const std::string &s) {
  return _hexa.validate(s);
}

