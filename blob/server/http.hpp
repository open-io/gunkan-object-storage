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
#include <memory>
#include <string>
#include <vector>
#include <mutex>  // NOLINT

#include "io.hpp"

#define BUFLEN 2048

#define HTTP_TIMEOUT_HEADERS 1000
#define HTTP_TIMEOUT_SEND_ERROR 5000
#define HTTP_TIMEOUT_SEND_HEADER 5000

#define CHUNK_FINAL "0\r\n\r\n"


struct HttpRequest;
struct HttpReply;

using Headers = std::map<std::string, std::string>;

enum class HttpRequestParsingStep {
  First = 0,
  HeaderName = 1,
  HeaderValue = 2,
  Body = 3,
};

enum class HttpMethod {
  Get = 0,
  Head = 1,
  Put = 2,
  Delete = 3,
  Copy = 4,
  Move = 5,
};


struct HttpRequest {
  std::shared_ptr<ActiveFD> client;
  Headers headers;
  std::shared_ptr<opentracing::Tracer> tracer;

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

  // Pending body
  std::vector<uint8_t> body;
  size_t body_offset;

  // Work in progress
  std::string hdr_field;
  std::string hdr_value;

  HttpRequestParsingStep step;
  http_parser parser;


  ~HttpRequest() {}

  HttpRequest() = delete;

  HttpRequest(HttpRequest &&o) = delete;

  HttpRequest(const HttpRequest &o) = delete;

  HttpRequest(
      std::shared_ptr<ActiveFD> c,
      std::shared_ptr<opentracing::Tracer> t);

  std::unique_ptr<HttpReply> make_reply();

  HttpMethod method() const;

  bool read_only() const;

  bool consume_headers(int64_t dl);

  void add_body(std::vector<uint8_t> b, size_t offset);

  /* Store in `headers` the key value pair stored in `hdr_field` and
   * `hdr`value`, then reset both fields. */
  void save_header();
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
