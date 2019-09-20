//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "internals.hpp"

RequestAcceptor::~RequestAcceptor() {
  if (fd_server >= 0) {
    ::close(fd_server);
    fd_server = -1;
  }
}

RequestAcceptor::RequestAcceptor(BlobServer *srv) :
  server{srv}, fd_server{-1} {}

dill_coroutine void RequestAcceptor::consume(int bundle) {
  const int cli_flags = SOCK_NONBLOCK|SOCK_CLOEXEC;
  BlobAddr addr{};
  std::deque<BlobClient*> pending;

  while (server->running()) {
    // Accept a batch of connections to avoid switching to another
    // thread or coroutine.
    for (int i{0}; i < BATCH_ACCEPTOR; ++i) {
      socklen_t len{sizeof(addr)};
      int fd_cli = ::accept4(fd_server, &addr.sa, &len, cli_flags);
      if (fd_cli < 0) {
        break;
      }
      pending.emplace_back(new BlobClient(server, fd_cli, addr));
    }

    if (pending.empty()) {
      (void) ::dill_fdin(fd_server, ::dill_now() + SLEEP_ACCEPTOR);
    } else {
      // start a coroutine for each connection
      while (!pending.empty()) {
        auto p = pending.front();
        pending.pop_front();
        dill_bundle_go(bundle, classify(p));
      }
    }
  }
}

static void _dump_request(HttpRequest *req __attribute__((unused))) {
#if 0
  DLOG(INFO)
    << http_method_str((enum http_method)req->parser.method)
    << " " << req->url;
  for (auto e : req->headers) {
    DLOG(INFO) << e.first << ": " << e.second;
  }
  DLOG(INFO) << "nread = " << req->parser.nread;
  DLOG(INFO) << "content length = " << req->parser.content_length;
  DLOG(INFO) << "chunked = " << static_cast<int>(req->parser.flags & F_CHUNKED);
#endif
}

dill_coroutine void RequestAcceptor::classify(HttpRequest *req) {
  int opt{0};
  const int fd = req->client->fd;
  const int64_t dl_headers = ::dill_now() + HTTP_TIMEOUT_HEADERS;

  auto s = req->tracer->StartSpan("hdr");
  bool rc = req->consume_headers(dl_headers);
  s->Finish();
  if (!rc)
    goto label_exit;

  _dump_request(req);

  // Classify now
  switch (req->parser.method) {
    case HTTP_PUT:
    case HTTP_COPY:
    case HTTP_MOVE:
    case HTTP_DELETE:
      opt = static_cast<int>(server->threads.executor_be_read.tos);
      setsockopt(fd, SOL_SOCKET, SO_PRIORITY, &opt, sizeof(opt));
      ::dill_fdclean(fd);
      return server->threads.executor_be_write.receive(req);

    case HTTP_GET:
    case HTTP_HEAD:
      opt = static_cast<int>(server->threads.executor_be_write.tos);
      setsockopt(fd, SOL_SOCKET, SO_PRIORITY, &opt, sizeof(opt));
      ::dill_fdclean(fd);
      return server->threads.executor_be_read.receive(req);

    default:
      HttpReply rep(req);
      rep.write_error(405);
  }

label_exit:
  ::dill_fdclean(fd);
  ::close(fd);
  DLOG(INFO) << "Acceptor -1 " << fd << " error";
}

dill_coroutine void RequestAcceptor::classify(BlobClient *c) {
  std::unique_ptr<std::ostringstream> output{new std::ostringstream{}};
  opentracing::mocktracer::MockTracerOptions options{};
  options.recorder = std::unique_ptr<opentracing::mocktracer::Recorder>{
      new opentracing::mocktracer::JsonRecorder{std::move(output)}};
  std::shared_ptr<opentracing::Tracer> tracer{
      new opentracing::mocktracer::MockTracer{std::move(options)}};

  std::shared_ptr<BlobClient> client(c);
  return classify(new HttpRequest(client, tracer));
}

