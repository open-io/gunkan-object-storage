//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_THREADS_HPP_
#define BLOB_SERVER_THREADS_HPP_

#include <libdill.h>

#include <deque>
#include <memory>

#include "io.hpp"
#include "http.hpp"
#include "blobid.hpp"


extern const int64_t one64;
extern bool flag_debug;
extern volatile bool flag_sys_running;


struct RequestExecutor;
struct RequestAcceptor;
struct ThreadRunner;


struct Thread {
  pthread_t th;

  template<class T>
  void start(const char *name, pthread_attr_t *attr);

  void join();
};


struct RequestExecutor : public Thread {
  HttpHandler *handler;
  int fd_tokens;

  int fd_wakeup;
  std::mutex latch;
  std::deque<std::unique_ptr<HttpRequest>> queue;

  IpTos tos;
  bool ioprio_rt;
  bool ioprio_high;

  ~RequestExecutor();

  RequestExecutor() = delete;

  RequestExecutor(RequestExecutor &&o) = delete;

  RequestExecutor(const RequestExecutor &o) = delete;

  explicit RequestExecutor(HttpHandler *h);

  dill_coroutine void consume(int bundle);

  /* Receive a pending Request from another worker */
  void receive(std::unique_ptr<HttpRequest> req);

  /* Forward the call to client->execute()
   * Utility method that helps transmitting the ownership of the unique pointer
   * to the request. */
  dill_coroutine void execute(std::unique_ptr<HttpRequest> req);
};


struct RequestAcceptor : public Thread {
  ThreadRunner *threads;
  int fd_server;

  ~RequestAcceptor();

  RequestAcceptor() = delete;

  RequestAcceptor(RequestAcceptor &&o) = delete;

  RequestAcceptor(const RequestAcceptor &o) = delete;

  explicit RequestAcceptor(ThreadRunner *th);

  dill_coroutine void consume(int bundle);

  /**
   * Validate, parse and forward the message to the most appropriated
   * worker.
   */
  dill_coroutine void classify(std::shared_ptr<ActiveFD> cli);
};


struct ThreadRunner {
  RequestAcceptor worker_ingress;
  RequestExecutor executor_be_read;
  RequestExecutor executor_be_write;
  RequestExecutor executor_rt_read;
  RequestExecutor executor_rt_write;

  ~ThreadRunner();

  ThreadRunner() = delete;

  ThreadRunner(const ThreadRunner &o) = delete;

  ThreadRunner(ThreadRunner &&o) = delete;

  explicit ThreadRunner(HttpHandler *h);

  void configure(int fd);

  void start();

  void stop();

  void join();
};


#endif  // BLOB_SERVER_THREADS_HPP_
