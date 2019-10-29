//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "threads.hpp"

#include <opentracing/mocktracer/tracer.h>
#include <opentracing/mocktracer/in_memory_recorder.h>

#include <sstream>

#define BATCH_ACCEPTOR 16
#define SLEEP_ACCEPTOR 1000


RequestAcceptor::~RequestAcceptor() {
  if (fd_server >= 0) {
    ::close(fd_server);
    fd_server = -1;
  }
}

RequestAcceptor::RequestAcceptor(ThreadRunner *th) :
  threads{th}, fd_server{-1} {}

dill_coroutine void RequestAcceptor::consume(int bundle) {
  const int cli_flags = SOCK_NONBLOCK | SOCK_CLOEXEC;
  std::deque<std::shared_ptr<ActiveFD>> pending;

  while (flag_sys_running) {
    bool eagain{false};

    // Accept a batch of connections to avoid switching to another
    // thread or coroutine.
    for (int i{0}; i < BATCH_ACCEPTOR; ++i) {
      NetAddr addr{};
      socklen_t len{sizeof(addr)};
      int fd_cli = ::accept4(fd_server, &addr.sa, &len, cli_flags);
      if (fd_cli < 0) {
        eagain = (errno == EAGAIN);
        break;
      }
      pending.emplace_back(new ActiveFD(fd_cli, addr));
    }

    // start a coroutine for each connection
    while (!pending.empty()) {
      auto p = pending.front();
      pending.pop_front();
      dill_bundle_go(bundle, classify(std::move(p)));
    }

    if (eagain)
      (void) ::dill_fdin(fd_server, ::dill_now() + SLEEP_ACCEPTOR);
  }
}

dill_coroutine void RequestAcceptor::classify(
    std::shared_ptr<ActiveFD> client) {
  // Prepare a tracing context
  opentracing::mocktracer::MockTracerOptions options{};
  options.recorder = std::unique_ptr<opentracing::mocktracer::Recorder>{
      new opentracing::mocktracer::InMemoryRecorder};

  std::shared_ptr<opentracing::Tracer> tracer{
      new opentracing::mocktracer::MockTracer{std::move(options)}};

  std::unique_ptr<HttpRequest> req(new HttpRequest(client, tracer));

  req->span_active = tracer->StartSpan("active");

  // Parse the headers
  req->span_parse = req->tracer->StartSpan("parse", {
    opentracing::ChildOf(&req->span_active->context())});
  bool rc = req->consume_headers(::dill_now() + HTTP_TIMEOUT_HEADERS);
  req->span_parse->Finish();

  client->detach();

  if (!rc) {
    req->span_active->Finish();
    return;
  }

  req->span_wait = tracer->StartSpan("wait", {
        opentracing::FollowsFrom(&req->span_parse->context()),
        opentracing::ChildOf(&req->span_active->context())});

  if (req->IsReadOnly()) {
    client->SetPrio(threads->executor_be_read.tos);
    return threads->executor_be_write.receive(std::move(req));
  } else {
    client->SetPrio(threads->executor_be_write.tos);
    return threads->executor_be_read.receive(std::move(req));
  }
}
