//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_HTTP_HPP_
#define BLOB_SERVER_HTTP_HPP_

#include <http_parser.h>
#include <opentracing/tracer.h>

#include <map>
#include <deque>
#include <memory>
#include <string>
#include <utility>
#include <vector>
#include <mutex>  // NOLINT

#include "io.hpp"
#include "bytes.hpp"

#define BUFLEN 2048

#define HTTP_TIMEOUT_HEADERS 1000
#define HTTP_TIMEOUT_SEND_ERROR 5000
#define HTTP_TIMEOUT_SEND_HEADER 5000

#define CHUNK_FINAL "0\r\n\r\n"


struct HttpRequest;
struct HttpReply;

using Headers = std::map<std::string, std::string>;

enum class HttpMethod {
  Get = 0,
  Head = 1,
  Put = 2,
  Delete = 3,
  Copy = 4,
  Move = 5,
};

class Accumulator {
 public:
  ~Accumulator() {}
  Accumulator(Accumulator &&o) = delete;
  Accumulator(const Accumulator &o) = delete;
  Accumulator() : ready_() {}
  void Add(Slice s) { ready_.push_back(s); }

 protected:
  ssize_t flush_pending(FileAppender &fa);

 protected:
  std::deque<Slice> ready_;
};

class BodyReader : public WriterTo, public Accumulator {
 public:
  ~BodyReader() {}
  BodyReader() : Accumulator() {}
  BodyReader(BodyReader &&o) = delete;
  BodyReader(const BodyReader &o) = delete;
};

class InlineBodyReader : public BodyReader {
 public:
  InlineBodyReader(const InlineBodyReader &o) = delete;

  InlineBodyReader(InlineBodyReader &&o) = delete;

  InlineBodyReader(ActiveFD &c, uint64_t l);

  ~InlineBodyReader();

  std::pair<int, uint64_t> WriteTo(FileAppender &fa) override;

 private:
  ssize_t transfer_zero_copy(FileAppender &fa);

 private:
  uint64_t total_;
  uint64_t consumed_;
  ActiveFD &client_;
};

class ChunkedBodyReader : public BodyReader {
 public:
  ChunkedBodyReader(const ChunkedBodyReader &o) = delete;

  ChunkedBodyReader(ChunkedBodyReader &&o) = delete;

  ChunkedBodyReader(ActiveFD &c, http_parser &p);

  ~ChunkedBodyReader();

  std::pair<int, uint64_t> WriteTo(FileAppender &fa) override;

 private:
  uint64_t consumed_;
  ActiveFD &client_;
  http_parser &parser_;
};

struct HttpRequest {
  std::shared_ptr<ActiveFD> client;
  Headers headers;
  std::shared_ptr<opentracing::Tracer> tracer;
  std::shared_ptr<BodyReader> body;

  std::string url;
  uint64_t bytes_in;
  uint64_t when_active;
  uint64_t when_parsed;
  uint64_t when_consumed;

  // Each request exposes 4 OpenTracing spans with the following hierarchy:
  // Active{Parse{} -> Wait{} -> Exec{}}
  std::unique_ptr<opentracing::Span> span_active;
  std::unique_ptr<opentracing::Span> span_parse;
  std::unique_ptr<opentracing::Span> span_wait;
  std::unique_ptr<opentracing::Span> span_exec;

  // Work in progress
  http_parser parser;


  ~HttpRequest() {}

  HttpRequest() = delete;

  HttpRequest(HttpRequest &&o) = delete;

  HttpRequest(const HttpRequest &o) = delete;

  HttpRequest(
      std::shared_ptr<ActiveFD> c,
      std::shared_ptr<opentracing::Tracer> t);

  std::unique_ptr<HttpReply> make_reply();

  HttpMethod GetMethod() const;

  bool IsReadOnly() const;

  bool consume_headers(int64_t dl);
};


struct HttpReply {
  std::shared_ptr<ActiveFD> client;
  Headers headers;
  std::shared_ptr<opentracing::Tracer> tracer;
  HttpRequest &req_;

  uint64_t bytes_out;
  uint64_t when_start;

  int64_t content_length_;
  int code_;

  ~HttpReply();

  HttpReply() = delete;

  HttpReply(const HttpReply &o) = delete;

  HttpReply(HttpReply &&o) = delete;

  HttpReply(
      HttpRequest &req,
      std::shared_ptr<ActiveFD> c,
      std::shared_ptr<opentracing::Tracer> t);

  void write_error(int code);

  bool write_headers(int code, int64_t content_length);

  bool write_chunk(uint8_t *base, size_t length);

  bool write_chunk(char *base, size_t length);

  bool write_final_chunk();
};

uint64_t total_microseconds(const HttpRequest &req, const HttpReply &rep);

uint64_t execution_microsends(const HttpRequest &req, const HttpReply &rep);

class HttpHandler {
 public:
  virtual void execute(
      std::unique_ptr<HttpRequest> req,
      std::unique_ptr<HttpReply> rep) = 0;
};


#endif  // BLOB_SERVER_HTTP_HPP_
