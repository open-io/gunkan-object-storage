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

enum class ParsingStep {
  First = 0,
  HeaderName = 1,
  HeaderValue = 2,
  Body = 3,
  Done = 4,
};

struct HeaderParsingContext {
  Headers headers;
  std::string hk, hv;
  std::string url;
  ParsingStep step;

  HeaderParsingContext(): headers(), hk{}, hv{},
                          url{}, step{ParsingStep::First} {}
  void save_header() { headers[std::move(hk)] = std::move(hv); }
};

static int _on_message_begin(http_parser *p) {
  HeaderParsingContext *ctx = static_cast<HeaderParsingContext*>(p->data);
  ctx->url.clear();
  return 0;
}

static int _on_url(http_parser *p, const char *b, size_t l) {
  HeaderParsingContext *ctx = static_cast<HeaderParsingContext*>(p->data);
  ctx->url.append(b, l);
  return 0;
}

static int _on_header_field(http_parser *p, const char *b, size_t l) {
  HeaderParsingContext *ctx = static_cast<HeaderParsingContext*>(p->data);
  switch (ctx->step) {
    case ParsingStep::HeaderValue:
      // A Header terminated, we keep the Key,Value for later
      assert(ctx->step == ParsingStep::HeaderValue);
      ctx->save_header();
      ctx->step = ParsingStep::First;
      // FALLTHROUGH
    case ParsingStep::First:
      // A new header starts
      assert(ctx->step == ParsingStep::First);
      ctx->hk.clear();
      ctx->hv.clear();
      ctx->step = ParsingStep::HeaderName;
      // FALLTHROUGH
    case ParsingStep::HeaderName:
      assert(ctx->step == ParsingStep::HeaderName);
      ctx->hk.append(b, l);
      return 0;
    default:
      SYSLOG(ERROR) << __func__ << "step=" << static_cast<int>(ctx->step);
      return -1;
  }
}

static int _on_header_value(http_parser *p, const char *b, size_t l) {
  HeaderParsingContext *ctx = static_cast<HeaderParsingContext*>(p->data);
  switch (ctx->step) {
    case ParsingStep::HeaderName:
      ctx->step = ParsingStep::HeaderValue;
      // FALLTHROUGH
    case ParsingStep::HeaderValue:
      ctx->hv.append(b, l);
      return 0;
    default:
      return -1;
  }
}

static int _on_headers_complete(http_parser *p) {
  HeaderParsingContext *ctx = static_cast<HeaderParsingContext*>(p->data);
  switch (ctx->step) {
    case ParsingStep::First:
      // Strange case, not header is present.
      // FALLTHROUGH
    case ParsingStep::HeaderValue:
      ctx->step = ParsingStep::Body;
      http_parser_pause(p, 1);
      return 0;
    default:
      return -1;
  }
}

static http_parser_settings cb_headers{
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
    client(c), headers(), tracer(t), body(),
    url(), bytes_in{0}, when_active{0}, when_parsed{0}, when_consumed{0},
    parser{} {
      when_active = monotonic_microseconds();
}

bool HttpRequest::consume_headers(int64_t dl) {
  http_parser_init(&parser, HTTP_REQUEST);
  HeaderParsingContext ctx;
  parser.data = &ctx;

  when_parsed = monotonic_microseconds();

  // Consume bytes, parse the request and stop at the end of the headers
  for (;;) {
    Slice r;
    if (0 != client->Read(&r, dl))
      return false;

    bytes_in += r.size();
    const size_t consumed = ::http_parser_execute(
        &parser, &cb_headers,
        reinterpret_cast<const char*>(r.data()), r.size());

    if (parser.http_errno == HPE_PAUSED) {
      if (parser.flags & F_CHUNKED) {
        body.reset(new ChunkedBodyReader(*client.get(), parser));
      } else {
#if 0
        body.reset(new InlineBodyReader(*client.get(), parser.content_length));
#else
        body.reset(new ChunkedBodyReader(*client.get(), parser));
#endif
      }
      if (consumed != r.size()) {
        r.skip(consumed);
        body->Add(std::move(r));
      }
      headers.swap(ctx.headers);
      url.swap(ctx.url);
      when_parsed = monotonic_microseconds();
      return true;
    }
    if (parser.http_errno != 0) {
      SYSLOG(INFO)
      << "HTTP ERROR "
      << http_errno_name((enum http_errno) parser.http_errno)
      << " " << http_errno_description((enum http_errno) parser.http_errno);
      return false;
    }

    assert(consumed == r.size());
  }
}

std::unique_ptr<HttpReply> HttpRequest::make_reply() {
  return std::unique_ptr<HttpReply>(new HttpReply(*this, client, tracer));
}

HttpMethod HttpRequest::GetMethod() const {
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

bool HttpRequest::IsReadOnly() const {
  switch (GetMethod()) {
    case HttpMethod::Put:
    case HttpMethod::Copy:
    case HttpMethod::Move:
    case HttpMethod::Delete:
      return false;
    default:
      return true;
  }
}

uint64_t total_microseconds(
    const HttpRequest &req,
    const HttpReply &rep __attribute__((unused))) {
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
  explicit ReplyWriter(Headers &h) : headers_{h} {}

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


ssize_t Accumulator::flush_pending(FileAppender &fa) {
  if (ready_.empty())
    return 0;
  std::vector<iovec> iov;
  for (auto &s : ready_)
    iov.emplace_back(iovec{s.data(), s.size()});
  if (!::_writev_full(fa.fd, iov.data(), iov.size(), ::dill_now() + 5000))
    return -1;
  auto s = compute_iov_size(iov.data(), iov.size());
  ready_.clear();
  return s;
}


InlineBodyReader::InlineBodyReader(ActiveFD &c, uint64_t l) :
  BodyReader(), total_{l}, consumed_{0}, client_{c} {}

InlineBodyReader::~InlineBodyReader() {}

std::pair<int, uint64_t> InlineBodyReader::WriteTo(FileAppender &fa) {
  fa.preallocate(total_);

  errno = 0;
  ssize_t w0 = flush_pending(fa);
  if (w0 < 0)
    return {errno, -1};
  consumed_ += w0;

  errno = 0;
  ssize_t w1 = transfer_zero_copy(fa);
  if (w1 < 0)
    return {errno, w0};
  consumed_ += w1;

  return {0, w0 + w1};
}

ssize_t InlineBodyReader::transfer_zero_copy(FileAppender &fa) {
  int64_t max = total_ - consumed_;
  int err = fa.splice(client_.fd, max);
  if (err == 0)
    return max;
  errno = err;
  return -1;
}


ChunkedBodyReader::~ChunkedBodyReader() {}

ChunkedBodyReader::ChunkedBodyReader(ActiveFD &c, http_parser &p) :
  BodyReader(), consumed_{0}, client_{c}, parser_{p} {}

struct BodyParsingContext {
  FileAppender &out;
  ParsingStep step;
  Slice current;
  std::vector<Slice> posted;
  size_t accumulated;

  ~BodyParsingContext() {}

  BodyParsingContext() = delete;

  BodyParsingContext(BodyParsingContext &&o) = delete;

  BodyParsingContext(const BodyParsingContext &o) = delete;

  explicit BodyParsingContext(FileAppender &fa) :
      out{fa}, step{ParsingStep::Body}, current{}, posted(), accumulated{0} {}

  void Add(Slice s) {
    accumulated += s.size();
    posted.push_back(std::move(s));
  }

  void Flush() {
    _writev_full(out.fd, posted, ::dill_now() + 10000);
    ::dill_yield();
    posted.clear();
    accumulated = 0;
  }
};

static int _on_message_end(http_parser *p) {
  BodyParsingContext *ctx = static_cast<BodyParsingContext*>(p->data);
  if (ctx->step != ParsingStep::Body)
    return -1;
  ctx->Flush();
  ctx->step = ParsingStep::Done;
  ::http_parser_pause(p, 1);
  return 0;
}

static int _on_body(http_parser *p, const char *b, size_t l) {
  BodyParsingContext *ctx = static_cast<BodyParsingContext*>(p->data);
  if (ctx->step != ParsingStep::Body)
    return -1;
  ctx->Add(ctx->current.sub(b, l));
  if (ctx->accumulated >= 8 * 1024 * 1024)
    ctx->Flush();
  return 0;
}

static http_parser_settings cb_body{
    .on_message_begin = nullptr,
    .on_url = nullptr,
    .on_status = nullptr,
    .on_header_field = nullptr,
    .on_header_value = nullptr,
    .on_headers_complete = nullptr,
    .on_body = _on_body,
    .on_message_complete = _on_message_end,
    .on_chunk_header = nullptr,
    .on_chunk_complete = nullptr,
};

class BlockMaker {
 private:
  size_t size = 32768, max = 8 * 1024 * 1024;
 public:
  size_t grow() {
    size <<= 1;
    size = size > max ? max : size;
    return size;
  }
  std::shared_ptr<Block> next() const {
    return std::shared_ptr<Block>(AllocatedBlock::Make(size));
  }
};

std::pair<int, uint64_t> ChunkedBodyReader::WriteTo(FileAppender &fa) {
  BlockMaker blockMaker;
  BodyParsingContext ctx(fa);
  uint64_t transferred{0};
  size_t accumulated{0};
  bool eof{false};

  ::http_parser_pause(&parser_, 0);
  parser_.data = &ctx;

  while (ctx.step != ParsingStep::Done) {
    if (!eof && ready_.empty()) {
      Slice s;
      auto block = blockMaker.next();
      int err = client_.ReadIn(block, &s, ::dill_now() + 30000);
      transferred += s.size();
      if (err == EOF)
        eof = true;
      else if (err)
        return {err, transferred};
      if (s.size() >= block->size())
        blockMaker.grow();
      accumulated += s.size();
      ready_.push_back(std::move(s));
    }
    assert(!eof || !ready_.empty());
    while (!ready_.empty()) {
      do {
        Slice s = ready_.front();
        ready_.pop_front();
        ctx.current = std::move(s);
      } while (0);

      const size_t consumed = ::http_parser_execute(
          &parser_, &cb_body,
          reinterpret_cast<const char *>(ctx.current.data()),
          ctx.current.size());
      accumulated -= consumed;

      if (consumed < ctx.current.size()) {
        ctx.current.skip(consumed);
        ready_.push_front(std::move(ctx.current));
      }
      if (parser_.http_errno == HPE_PAUSED)
        break;
      if (parser_.http_errno != 0) {
        LOG(ERROR) << __func__ << " http error "
                   << ::http_errno_description((http_errno) parser_.http_errno);
        return {EBADMSG, transferred};
      }
    }
  }

  assert(ctx.step == ParsingStep::Done);
  if (!ready_.empty())
    SYSLOG(ERROR) << "HTTP Request: additionnal data detected";
  errno = 0;
  return {0, transferred};
}
