//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "threads.hpp"

#include <unistd.h>
#include <fcntl.h>
#include <signal.h>
#include <sys/socket.h>
#include <sys/eventfd.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/resource.h>

#include <gflags/gflags.h>
#include <glog/logging.h>

#include <cassert>
#include <fstream>
#include <iostream>
#include <map>
#include <memory>
#include <mutex>  // NOLINT
#include <string>
#include <utility>
#include <vector>

#define THREAD_STACK_SIZE 16384

#define CALL(Fn, ...) do { \
  int rc = Fn(__VA_ARGS__); \
  if (rc != 0) { \
    DLOG(INFO) << #Fn \
      << " rc=" << rc << "/" << ::strerror(rc) \
      << " errno=" << errno << "/" << ::strerror(errno); \
  } \
  assert(rc == 0); \
} while (0)


static uint8_t stack_ingress[THREAD_STACK_SIZE];
static uint8_t stack_be_read[THREAD_STACK_SIZE];
static uint8_t stack_be_write[THREAD_STACK_SIZE];
static uint8_t stack_rt_read[THREAD_STACK_SIZE];
static uint8_t stack_rt_write[THREAD_STACK_SIZE];

static void _block_and_ignore_signals()
  __attribute__((cold));

static void _block_and_ignore_signals() {
  sigset_t new_set{}, old_set{};

  sigemptyset(&new_set);
  sigemptyset(&old_set);
  for (auto sig : {SIGINT, SIGTERM, SIGUSR1, SIGUSR2, SIGHUP, SIGPIPE})
    sigaddset(&new_set, sig);
  pthread_sigmask(SIG_BLOCK, &new_set, &old_set);
  sigprocmask(SIG_BLOCK, &new_set, &old_set);
}

template <class T>
static void * _worker(void *p) {
  ::dill_bundle_storage storage{};

  _block_and_ignore_signals();

  int handle = ::dill_bundle_mem(&storage);
  if (handle < 0) {
    SYSLOG(ERROR) << "libdill bundle creation failure: " << ::strerror(errno);
  } else {
    dill_bundle_go(handle, static_cast<T *>(p)->consume(handle));
    if (0 != ::dill_bundle_wait(handle, -1)) {
      SYSLOG(ERROR) << "libdill coroutines are still running";
    }
    ::dill_hclose(handle);
  }
  return p;
}

template <class T>
void Thread::start(const char *name, pthread_attr_t *attr) {
  CALL(pthread_create, &th, attr, _worker<T>, this);
  pthread_setname_np(th, name);
}

void Thread::join() {
  (void) pthread_join(th, nullptr);
}

ThreadRunner::~ThreadRunner() {}

ThreadRunner::ThreadRunner(HttpHandler *h) :
  worker_ingress(this),
  executor_be_read(h), executor_be_write(h),
  executor_rt_read(h), executor_rt_write(h) {}

void ThreadRunner::configure(int fd) {
  worker_ingress.fd_server = fd;
}

void ThreadRunner::start() {
  pthread_attr_t th_attr{};
  sched_param param{};
  struct rlimit rlim{};

  bool run_as_root = (getuid() == 0);

  // The workers are really lean and are made to work with minimal stacks
  rlim.rlim_cur = rlim.rlim_max = THREAD_STACK_SIZE;
  setrlimit(RLIMIT_STACK, &rlim);

  // All the threads of a process must share the same scheduling policy
  // and may only have the priority as a variable. It also requires that
  // the process is running as root.
  CALL(sched_getparam, 0, &param);
  if (!run_as_root) {
    SYSLOG(WARNING) << "Running as non-root, no QoS possible";
  } else {
    param.sched_priority = 1;
    CALL(sched_setscheduler, 0, SCHED_RR, &param);
  }

  CALL(pthread_attr_init, &th_attr);
  CALL(pthread_attr_setguardsize, &th_attr, 0);
  CALL(pthread_attr_setstacksize, &th_attr, THREAD_STACK_SIZE);

  // Start the executor for READ
  // We give it a medium priority, so that consuming the data will happen
  // before the acceptance of connections and qualifications of requests,
  // but less than the ingestion of data.
  // Start the executor for WRITE
  // We give it the highest priority to be always able to ingest data.


  // Gold class ---------------------------------------------
  // 2 workers for read/write
  // * higher CPU priority
  // * real-time I/O priority
  // --------------------------------------------------------
  CALL(pthread_attr_setdetachstate, &th_attr, PTHREAD_CREATE_JOINABLE);
  if (run_as_root) {
    CALL(pthread_attr_setinheritsched, &th_attr, PTHREAD_EXPLICIT_SCHED);
    CALL(pthread_attr_setschedpolicy, &th_attr, SCHED_RR);
    param.sched_priority = 5;
    CALL(pthread_attr_setschedparam, &th_attr, &param);
  }
  executor_rt_read.ioprio_rt = true;
  executor_rt_read.ioprio_high = false;
  executor_rt_write.tos = IpTos::Throughput;
  pthread_attr_setstack(&th_attr, stack_rt_read, THREAD_STACK_SIZE);
  executor_rt_read.start<RequestExecutor>("ig-blob@read-RT", &th_attr);

  CALL(pthread_attr_setdetachstate, &th_attr, PTHREAD_CREATE_JOINABLE);
  if (run_as_root) {
    CALL(pthread_attr_setinheritsched, &th_attr, PTHREAD_EXPLICIT_SCHED);
    CALL(pthread_attr_setschedpolicy, &th_attr, SCHED_RR);
    param.sched_priority = 6;
    CALL(pthread_attr_setschedparam, &th_attr, &param);
  }
  executor_rt_write.ioprio_rt = true;
  executor_rt_write.ioprio_high = true;
  executor_rt_write.tos = IpTos::Throughput;
  pthread_attr_setstack(&th_attr, stack_rt_write, THREAD_STACK_SIZE);
  executor_rt_write.start<RequestExecutor>("ig-blob@write-RT", &th_attr);


  // Best-Effort class --------------------------------------
  // 2 workers for read/write
  // * average CPU priority
  // * best-effort I/O priority
  // --------------------------------------------------------

  CALL(pthread_attr_setdetachstate, &th_attr, PTHREAD_CREATE_JOINABLE);
  if (run_as_root) {
    CALL(pthread_attr_setinheritsched, &th_attr, PTHREAD_EXPLICIT_SCHED);
    CALL(pthread_attr_setschedpolicy, &th_attr, SCHED_RR);
    param.sched_priority = 3;
    CALL(pthread_attr_setschedparam, &th_attr, &param);
  }
  executor_be_read.ioprio_rt = false;
  executor_be_read.ioprio_high = false;
  executor_be_read.tos = IpTos::LowCost;
  pthread_attr_setstack(&th_attr, stack_be_read, THREAD_STACK_SIZE);
  executor_be_read.start<RequestExecutor>("ig-blob@read-BE", &th_attr);

  CALL(pthread_attr_setdetachstate, &th_attr, PTHREAD_CREATE_JOINABLE);
  if (run_as_root) {
    CALL(pthread_attr_setinheritsched, &th_attr, PTHREAD_EXPLICIT_SCHED);
    CALL(pthread_attr_setschedpolicy, &th_attr, SCHED_RR);
    param.sched_priority = 4;
    CALL(pthread_attr_setschedparam, &th_attr, &param);
  }
  executor_be_write.ioprio_rt = false;
  executor_be_write.ioprio_high = true;
  executor_be_write.tos = IpTos::LowCost;
  pthread_attr_setstack(&th_attr, stack_be_write, THREAD_STACK_SIZE);
  executor_be_write.start<RequestExecutor>("ig-blob@write-BE", &th_attr);

  // Start the worker_ingress (that also qualifies the requests):
  // We give it a low priority, just below the main thread for signals
  CALL(pthread_attr_setdetachstate, &th_attr, PTHREAD_CREATE_JOINABLE);
  if (run_as_root) {
    CALL(pthread_attr_setinheritsched, &th_attr, PTHREAD_EXPLICIT_SCHED);
    CALL(pthread_attr_setschedpolicy, &th_attr, SCHED_RR);
    param.sched_priority = 2;
    CALL(pthread_attr_setschedparam, &th_attr, &param);
  }
  pthread_attr_setstack(&th_attr, stack_ingress, THREAD_STACK_SIZE);
  worker_ingress.start<RequestAcceptor>("ig-blob@ingress", &th_attr);
}

void ThreadRunner::stop() { flag_sys_running = false; }

void ThreadRunner::join() {
  worker_ingress.join();
  executor_be_write.join();
  executor_rt_write.join();
  executor_be_read.join();
  executor_rt_read.join();
}
