//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "internals.hpp"

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


HttpRequest::HttpRequest(std::shared_ptr<BlobClient> c,
    std::shared_ptr<opentracing::Tracer> t) :
    client(c), url(), headers(), tracer(t),
    body(), body_offset{0},
    hdr_field(), hdr_value(),
    step{HttpRequestParsingStep::First}, parser{} {
  when_active = microseconds();
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
  when_headers = microseconds();
  return true;
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
    case 418: return "No Such Handler";
    case 500: return "Internal Error";
    case 501: return "Not Implemented";
    case 502: return "Backend Error";
    case 503: return "Busy";
    default: return "Wot";
  }
}

static std::string _pack_reply_header(int code, int64_t content_length) {
  std::stringstream ss;
  ss << "HTTP/1.1 " << code << ' ' << _code2msg(code) << "\r\n"
    << "Connection: close\r\n";
  if (content_length >= 0) {
    ss << "Content-Length: " << content_length << "\r\n";
  } else {
    ss << "Transfer-Encoding: chunked\r\n";
  }
  ss << "\r\n";
  return ss.str();
}

HttpReply::~HttpReply() {
  int64_t when_final = microseconds();
  SYSLOG(INFO) << "access " << request->url
    << ' ' << (request->when_active - request->client->when_accept)
    << ' ' << (request->when_headers - request->when_active)
    << ' ' << (when_final - request->client->when_accept);
}

HttpReply::HttpReply(HttpRequest *req) : client(req->client), request{req} {}

bool HttpReply::write_error(int code) {
  return write_headers(code, 0);
}

bool HttpReply::write_headers(int code, int64_t content_length) {
  std::string s = _pack_reply_header(code, content_length);
  return _write_full(client->fd,
                     reinterpret_cast<uint8_t *>(s.data()), s.size(),
                     ::dill_now() + HTTP_TIMEOUT_SEND_HEADER);
}

bool HttpReply::write_final_chunk() {
  return SEND(client->fd, CHUNK_FINAL);
}

bool HttpReply::write_chunk(uint8_t *base, size_t length) {
  static char CRLF[] = "\r\n";

  std::stringstream ss;
  ss << length << "\r\n";
  auto s = ss.str();

  struct iovec iov[3];
  iov[0].iov_base = const_cast<char*>(s.data());
  iov[0].iov_len = s.size();
  iov[1].iov_base = base;
  iov[1].iov_len = length;
  iov[2].iov_base = CRLF;
  iov[2].iov_len = 2;
  return _writev_full(client->fd, iov, 3, ::dill_now() + 1000);
}

bool HttpReply::write_chunk(char *base, size_t length) {
  return write_chunk(reinterpret_cast<uint8_t*>(base), length);
}

