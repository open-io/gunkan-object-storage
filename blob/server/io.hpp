//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_IO_HPP_
#define BLOB_SERVER_IO_HPP_

#include <sys/types.h>
#include <arpa/inet.h>
#include <netinet/in.h>

#include <cstdint>
#include <vector>
#include <utility>
#include <memory>

#include "bytes.hpp"

enum class IpTos {
  Default = 0,
  LowCost = 1,
  Reliability = 2,
  Throughput = 3,
  LowDelay = 4,
  Precedence0 = 5,
  Precedence1 = 6,
  Precedence2 = 7,
};


union NetAddr {
  struct sockaddr sa;
  struct sockaddr_in sin;
  struct sockaddr_in6 sin6;
};


struct FD {
  int fd;

  virtual ~FD();
  FD();
  FD(const FD &o) = delete;
  FD(FD &&o) = delete;
  bool valid() const;
  explicit FD(int f);
  void detach();
  void close();
};

struct FileAppender : public FD {
  int64_t written;
  int64_t allocated;
  bool extend_allowed;

  ~FileAppender() override;
  FileAppender();
  explicit FileAppender(int f);

  void preallocate(int64_t size);
  int truncate();
  int splice(int from, int64_t size);
};

class WriterTo {
 public:
  /**
   * Perform a zero-copy data transfer from the current Reader to the
   * @param fa the target of the data transfer
   * @return true and the size that has been transferred upon success, or
   *         false and a number of bytes actually transferred in case of error
   *         (and errno is set).
   */
  virtual std::pair<int, uint64_t> WriteTo(FileAppender &fa) = 0;
};

struct ActiveFD : public FD {
  NetAddr peer;

  ~ActiveFD() override;

  ActiveFD() = delete;

  ActiveFD(ActiveFD &&o) = delete;

  ActiveFD(const ActiveFD &o) = delete;

  ActiveFD(int f, const NetAddr &a);

  void SetPrio(IpTos tos);

  int Sendfile(int from, int64_t size);

  int Read(Slice *out, int64_t dl);

  int ReadIn(std::shared_ptr<Block> block, Slice *out, int64_t dl);
};


class Pipe {
 private:
  int fd[2];
 public:
  ~Pipe();

  Pipe();

  Pipe(Pipe &&o) = delete;

  Pipe(const Pipe &o) = delete;

  bool init();

  int head() const;

  int tail() const;
};

size_t compute_iov_size(iovec *iov, size_t nb);

ssize_t _read_at_least(int fd, uint8_t *base, size_t max, size_t min,
    int64_t dl);

bool _write_full(int fd, const uint8_t *buf, size_t len, int64_t dl);

bool _writev_full(int fd, iovec *iov, size_t len, int64_t dl);

bool _writev_full(int fd, std::vector<iovec> &iov, int64_t dl);

bool _writev_full(int fd, std::vector<Slice> &slices, int64_t dl);

#endif  // BLOB_SERVER_IO_HPP_
