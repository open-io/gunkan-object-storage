//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "http.hpp"

#include <libdill.h>
#include <glog/logging.h>
#include <opentracing/tracer.h>
#include <opentracing/propagation.h>
#include <opentracing/string_view.h>

#include <string>
#include <utility>
#include <ios>

#include "io.hpp"
#include "time.hpp"
#include "strings.hpp"

#define SEND(FD, M) _write_full(FD, \
  reinterpret_cast<const uint8_t *>(M), sizeof(M)-1, \
  ::dill_now() + HTTP_TIMEOUT_SEND_ERROR)

static int _on_message_begin(http_parser *p) {
  HttpRequest *req = static_cast<HttpRequest*>(p->data);
  req->url.clear();
  return 0;
}

static int _on_url(http_parser *p, const char *b, size_t l) {
  HttpRequest *req = static_cast<HttpRequest*>(p->data);
  req->url.append(b, l);
  return 0;
}

static int _on_header_field(http_parser *p, const char *b, size_t l) {
  HttpRequest *req = static_cast<HttpRequest*>(p->data);
  switch (req->step) {
    case HttpRequestParsingStep::HeaderValue:
      // A Header terminated, we keep the Key,Value for later
      req->save_header();
      // FALLTHROUGH
    case HttpRequestParsingStep::First:
      // A new header starts
      req->step = HttpRequestParsingStep::HeaderName;
      req->hdr_field.clear();
      req->hdr_value.clear();
      // FALLTHROUGH
    case HttpRequestParsingStep::HeaderName:
      req->hdr_field.append(b, l);
      return 0;
    default:
      return -1;
  }
}

static int _on_header_value(http_parser *p, const char *b, size_t l) {
  HttpRequest *req = static_cast<HttpRequest*>(p->data);
  switch (req->step) {
    case HttpRequestParsingStep::HeaderName:
      req->step = HttpRequestParsingStep::HeaderValue;
      // FALLTHROUGH
    case HttpRequestParsingStep::HeaderValue:
      req->hdr_value.append(b, l);
      return 0;
    default:
      return -1;
  }
}

static int _on_headers_complete(http_parser *p) {
  HttpRequest *req = static_cast<HttpRequest*>(p->data);
  switch (req->step) {
    case HttpRequestParsingStep::First:
      // Strange case, not header is present.
      // FALLTHROUGH
    case HttpRequestParsingStep::HeaderValue:
      req->step = HttpRequestParsingStep::Body;
      http_parser_pause(&req->parser, 1);
      return 0;
    default:
      return -1;
  }
}

http_parser_settings settings{
    .on_message_begin = _on_message_begin,
    .on_url = _on_url,
    .on_status = nullptr,
    .on_header_field = _on_header_field,
    .on_header_value = _on_header_value,
    .on_headers_complete = _on_headers_complete,
    .on_body = nullptr,
    .on_message_complete = nullptr,
    .on_chunk_header = nullptr,
    .on_chunk_complete = nullptr,
};


HttpRequest::HttpRequest(
    std::shared_ptr<ActiveFD> c,
    std::shared_ptr<opentracing::Tracer> t) :
    client(c), headers(), tracer(t),
    url(), bytes_in{0}, when_active{0}, when_parsed{0}, when_consumed{0},
    body(), body_offset{0},
    hdr_field(), hdr_value(),
    step{HttpRequestParsingStep::First}, parser{} {
      when_active = monotonic_microseconds();
}

void HttpRequest::add_body(std::vector<uint8_t> b, size_t offset) {
  body.swap(b);
  body_offset = offset;
}

void HttpRequest::save_header() {
  headers.emplace(std::move(hdr_field), std::move(hdr_value));
}

bool HttpRequest::consume_headers(int64_t dl) {
  size_t filled{0};
  size_t consumed{0};
  std::vector<uint8_t> buffer(BUFLEN);

  http_parser_init(&parser, HTTP_REQUEST);
  parser.data = this;

  when_parsed = monotonic_microseconds();

  // Consume bytes, parse the request and stop at the end of the headers
  assert(step == HttpRequestParsingStep::First);
  while (step != HttpRequestParsingStep::Body) {
    consumed = 0;
    ssize_t r = ::recv(client->fd, buffer.data(), buffer.size(), MSG_NOSIGNAL);
    if (r < 0) {
      if (errno == EINTR)
        continue;
      if (errno == EAGAIN) {
        if (!::dill_fdin(client->fd, dl))
          continue;
        SYSLOG(INFO) << "TIMEOUT";
      } else {
        SYSLOG(INFO) << "NET ERROR " << errno << " " << ::strerror(errno);
      }
      return false;
    }

    bytes_in += r;
    filled = r;
    consumed = ::http_parser_execute(&parser, &settings,
        reinterpret_cast<const char*>(buffer.data()), filled);
    if (parser.http_errno != 0) {
      if (parser.http_errno != HPE_PAUSED) {
        SYSLOG(INFO) << "HTTP ERROR"
          << " " << http_errno_name((enum http_errno)parser.http_errno)
          << " " << http_errno_description((enum http_errno)parser.http_errno);
        return false;
      }
    }

    if (consumed != filled) {
      if (step != HttpRequestParsingStep::Body) {
        SYSLOG(INFO) << "BUG";
        return false;
      }
    }
  }

  if (consumed < filled) {
    buffer.resize(filled);
    add_body(std::move(buffer), consumed);
  }

  http_parser_pause(&parser, 0);
  when_parsed = monotonic_microseconds();
  return true;
}

std::unique_ptr<HttpReply> HttpRequest::make_reply() {
  return std::unique_ptr<HttpReply>(new HttpReply(*this, client, tracer));
}

HttpMethod HttpRequest::method() const {
  switch (parser.method) {
    case HTTP_PUT:
      return HttpMethod::Put;
    case HTTP_COPY:
      return HttpMethod::Copy;
    case HTTP_MOVE:
      return HttpMethod::Move;
    case HTTP_DELETE:
      return HttpMethod::Delete;
    case HTTP_HEAD:
      return HttpMethod::Head;
    case HTTP_GET:
    default:
      return HttpMethod::Get;
  }
}

bool HttpRequest::read_only() const {
  switch (method()) {
    case HttpMethod::Put:
    case HttpMethod::Copy:
    case HttpMethod::Move:
    case HttpMethod::Delete:
      return false;
    default:
      return true;
  }
}

uint64_t total_microseconds(const HttpRequest &req, const HttpReply &rep) {
  const uint64_t pre = req.when_active;
  const uint64_t post = monotonic_microseconds();
  return post - pre;
}

uint64_t execution_microsends(const HttpRequest &req, const HttpReply &rep) {
  const uint64_t pre = req.when_parsed;
  const uint64_t post = rep.when_start;
  return post - pre;
}

static const char * _code2msg(int code) {
  switch (code) {
    case 200: return "OK";
    case 201: return "Created";
    case 202: return "Accepted";
    case 204: return "No Content";
    case 206: return "Partial Content";
    case 400: return "Bad Request";
    case 403: return "Forbidden";
    case 404: return "Not Found";
    case 405: return "Method Not Allowed";
    case 408: return "Timeout";
    case 409: return "Conflict";
    case 418: return "No Such Handler";
    case 499: return "Client error";
    case 500: return "Internal Error";
    case 501: return "Not Implemented";
    case 502: return "Backend Error";
    case 503: return "Busy";
    default: return "Wot";
  }
}

struct ReplyWriter : opentracing::HTTPHeadersWriter {
  ReplyWriter(Headers &h) : headers_{h} {}

  opentracing::expected<void> Set(
      opentracing::string_view key,
      opentracing::string_view value) const override {
    headers_[key] = value;
    return {};
 }

  Headers &headers_;
};

static std::string _pack_reply_header(
    int code,
    int64_t content_length,
    HttpRequest *req,
    HttpReply *rep) {
  std::stringstream ss;

  // Extract the OpenTracing headers
  Headers ot_headers;
  ReplyWriter w(ot_headers);
  rep->tracer->Inject(req->span_active->context(), w);

  ss << "HTTP/1.1 " << code << ' ' << _code2msg(code) << CRLF;
  ss << "Connection: close" << CRLF;
  if (content_length >= 0)
    ss << "Content-Length: " << content_length << CRLF;
  else
    ss << "Transfer-Encoding: chunked" << CRLF;
  for (const auto &e : rep->headers)
    ss << e.first << ": " << e.second << CRLF;
  for (const auto &e : ot_headers)
    ss << e.first << ": " << e.second << CRLF;
  ss << CRLF;
  return ss.str();
}

HttpReply::~HttpReply() {}

HttpReply::HttpReply(
    HttpRequest &req,
    std::shared_ptr<ActiveFD> c,
    std::shared_ptr<opentracing::Tracer> t) : client(c), tracer(t), req_{req} {}

void HttpReply::write_error(int code) {
  write_headers(code, 0);
}

bool HttpReply::write_headers(int code, int64_t content_length) {
  std::string s = _pack_reply_header(code, content_length, &req_, this);
  bytes_out += s.size();
  when_start = monotonic_microseconds();
  code_ = code;
  return _write_full(client->fd,
                     reinterpret_cast<uint8_t *>(s.data()), s.size(),
                     ::dill_now() + HTTP_TIMEOUT_SEND_HEADER);
}

bool HttpReply::write_final_chunk() {
  auto rc = SEND(client->fd, CHUNK_FINAL);
  bytes_out += sizeof(CHUNK_FINAL) - 1;
  return rc;
}

bool HttpReply::write_chunk(uint8_t *base, size_t length) {
  static char CRLF[] = "\r\n";

  std::stringstream ss;
  ss << std::hex << length << "\r\n";
  auto s = ss.str();

  struct iovec iov[3];
  iov[0].iov_base = s.data();
  iov[0].iov_len = s.size();
  iov[1].iov_base = base;
  iov[1].iov_len = length;
  iov[2].iov_base = CRLF;
  iov[2].iov_len = 2;

  bytes_out += compute_iov_size(iov, 3);
  return _writev_full(client->fd, iov, 3, ::dill_now() + 1000);
}

bool HttpReply::write_chunk(char *base, size_t length) {
  return write_chunk(reinterpret_cast<uint8_t*>(base), length);
}
