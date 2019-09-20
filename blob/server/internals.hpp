//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_INTERNALS_HPP_
#define BLOB_SERVER_INTERNALS_HPP_

#include <getopt.h>
#include <unistd.h>
#include <signal.h>
#include <pthread.h>
#include <poll.h>
#include <fcntl.h>
#include <sys/resource.h>
#include <sys/socket.h>
#include <sys/eventfd.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/uio.h>
#include <arpa/inet.h>
#include <netinet/in.h>

#if 0
# ifdef __linux
#  include <sys/syscall.h>
#  include <linux/ioprio.h>
# endif
#endif

#include <glog/logging.h>
#include <gflags/gflags.h>
#include <libdill.h>
#include <http_parser.h>
#include <opentracing/tracer.h>
#include <opentracing/mocktracer/tracer.h>
#include <opentracing/mocktracer/json_recorder.h>

#include <cassert>
#include <deque>
#include <fstream>
#include <iostream>
#include <map>
#include <memory>
#include <mutex>  // NOLINT
#include <string>
#include <utility>
#include <vector>

#define NS_CHUNK_SIZE 64 * 1024 * 1024

#define BATCH_ACCEPTOR 16
#define SLEEP_ACCEPTOR 1000

#define BATCH_EXECUTOR 16
#define SLEEP_EXECUTOR 1000

#define THREAD_STACK_SIZE 16384
#define CORO_STACK_SIZE 16384

#define ALIGN 8
#define BUFLEN 2048

#define HTTP_TIMEOUT_HEADERS 1000
#define HTTP_TIMEOUT_SEND_ERROR 5000
#define HTTP_TIMEOUT_SEND_HEADER 5000

#define URL_PREFIX_INFO "/info"
#define URL_PREFIX_V1_BLOB "/v1/blob/"
#define URL_PREFIX_V1_STATUS "/v1/status"

#define CALL(Fn, ...) do { \
  int rc = Fn(__VA_ARGS__); \
  if (rc != 0) { \
    DLOG(INFO) << #Fn \
      << " rc=" << rc << "/" << ::strerror(rc) \
      << " errno=" << errno << "/" << ::strerror(errno); \
  } \
  assert(rc == 0); \
} while (0)


#define CHUNK_FINAL \
  "0\r\n" \
  "\r\n"

// ---------------------------------------------------------------------------

static inline uint8_t * _align(uint8_t *buf)
__attribute__((hot, pure, const));

static inline uint8_t * _align(uint8_t *buf) {
  return buf + ALIGN - (reinterpret_cast<ulong>(buf) % ALIGN);
}

ssize_t _read_at_least(int fd, uint8_t *base, size_t max, size_t min,
    int64_t dl);

bool _write_full(int fd, const uint8_t *buf, size_t len, int64_t dl);

bool _writev_full(int fd, struct iovec *iov, size_t len, int64_t dl);

bool _starts_with(const std::string &s, const std::string &p);

int64_t microseconds();

bool is_hexa(const std::string &s);

// ---------------------------------------------------------------------------

extern const int64_t one64;

// ---------------------------------------------------------------------------

union BlobAddr;
struct BlobConfig;
struct BlobStats;
struct BlobService;
struct BlobClient;
struct ThreadRunner;
struct RequestAcceptor;
struct RequestExecutor;

struct BlobServer;

struct HttpRequest;
struct HttpReply;


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


union BlobAddr {
  struct sockaddr sa;
  struct sockaddr_in sin;
  struct sockaddr_in6 sin6;
};


struct BlobId {
  std::string id_content;
  std::string id_part;
  unsigned int position;

  std::string encode() const;
  bool decode(const std::string &s);
  static bool decode(BlobId &id, const std::string &s) {
    return id.decode(s);
  }
};

struct BlobConfig {
  // Parameters of the file hierarchy on the backend filesystem
  unsigned int hash_width;
  unsigned int hash_depth;

  // Number of connections being qualified
  unsigned int workers_ingress;

  // Number of connections managed by the threads of the Best-Effort pool
  unsigned int workers_be_read;
  unsigned int workers_be_write;

  // Number of connections managed by the threads of the Real-Time pool
  unsigned int workers_rt_read;
  unsigned int workers_rt_write;

  // Flags that control the main process
  bool flag_help;
  bool flag_verbose;
  bool flag_quiet;
  bool flag_daemonize;
  bool flag_initiate;

  // Configuration directives with an impact on the main routine
  std::string argv0;
  std::string endpoint;
  std::string nsname;
  std::string pidfile;
  std::string basedir;

  ~BlobConfig() {}

  BlobConfig(const BlobConfig &o) = delete;

  BlobConfig(BlobConfig &&o) = delete;

  BlobConfig(): hash_width{2}, hash_depth{2},
                workers_ingress{64},
                workers_be_read{1024}, workers_be_write{1024},
                workers_rt_read{8}, workers_rt_write{8},
                flag_help{false}, flag_verbose{false},
                flag_daemonize{false}, flag_initiate{false},
                argv0(), endpoint(), nsname(), pidfile(), basedir() {}
};


struct BlobClient {
  int fd;
  BlobAddr peer;
  BlobServer *server;
  int64_t when_accept;

  ~BlobClient() {}
  BlobClient() = delete;
  BlobClient(BlobClient &&o) = delete;
  BlobClient(const BlobClient &o) = delete;
  BlobClient(BlobServer *s, int f, const BlobAddr &a);
};


struct Thread {
  pthread_t th;

  template <class T> void start(const char *name, pthread_attr_t *attr);
  void join();
};


struct RequestExecutor : public Thread {
  BlobServer *server;
  int fd_tokens;

  int fd_wakeup;
  std::mutex latch;
  std::deque<HttpRequest*> queue;

  IpTos tos;
  bool ioprio_rt;
  bool ioprio_high;

  ~RequestExecutor();
  RequestExecutor() = delete;
  RequestExecutor(RequestExecutor &&o) = delete;
  RequestExecutor(const RequestExecutor &o) = delete;
  explicit RequestExecutor(BlobServer *srv);

  dill_coroutine void consume(int bundle);

  /* Receive a pending Request from another worker */
  void receive(HttpRequest* req);

  /* Forward the call to client->execute()
   * Utility method that helps transmitting the ownership of the unique pointer
   * to the request. */
  dill_coroutine void execute(HttpRequest *req);
};


struct RequestAcceptor : public Thread {
  BlobServer *server;
  int fd_server;

  ~RequestAcceptor();
  RequestAcceptor() = delete;
  RequestAcceptor(RequestAcceptor &&o) = delete;
  RequestAcceptor(const RequestAcceptor &o) = delete;
  explicit RequestAcceptor(BlobServer *srv);

  dill_coroutine void consume(int bundle);

  /**
   * Validate, parse and forward the message to the most appropriated
   * worker.
   */
  dill_coroutine void classify(BlobClient *cli);

  void classify(HttpRequest *req);
};


struct BlobStats {
  uint64_t time_status;
  uint64_t time_put;
  uint64_t time_get;
  uint64_t time_delete;

  uint64_t hits_status;
  uint64_t hits_put;
  uint64_t hits_get;
  uint64_t hits_delete;

  // Counter of response codes

  uint64_t count_200;
  uint64_t count_201;
  uint64_t count_204;
  uint64_t count_206;
  uint64_t count_400;
  uint64_t count_403;
  uint64_t count_404;
  uint64_t count_405;
  uint64_t count_418;  // Returned in case of bad route
  uint64_t count_503;
  uint64_t count_50X;
};


struct ThreadRunner {
  BlobServer *server;
  volatile bool flag_running;

  RequestAcceptor worker_ingress;
  RequestExecutor executor_be_read;
  RequestExecutor executor_be_write;
  RequestExecutor executor_rt_read;
  RequestExecutor executor_rt_write;

  ~ThreadRunner();
  ThreadRunner() = delete;
  ThreadRunner(const ThreadRunner &o) = delete;
  ThreadRunner(ThreadRunner &&o) = delete;
  explicit ThreadRunner(BlobServer *srv);

  void stop();
  bool running() const;
  void start();
  void join();
};


struct BlobService {
  BlobServer *server;

  ~BlobService();
  BlobService() = delete;
  BlobService(BlobService &&o) = delete;
  BlobService(const BlobService &o) = delete;
  explicit BlobService(BlobServer *s);

  /**
   * Build the fullpath of a blob from the given ID extracted from
   * the request.
   */
  std::string fullpath(const BlobId &id);

  void execute(HttpRequest *req, HttpReply *rep);

  bool do_info(HttpRequest *req, HttpReply *rep);
  bool do_v1_blob(HttpRequest *req, HttpReply *rep);
  bool do_v1_status(HttpRequest *req, HttpReply *rep);

  bool do_v1_blob_put(HttpRequest *req, HttpReply *rep);
  bool do_v1_blob_put_inline(HttpRequest *req, HttpReply *rep, int fd);
  bool do_v1_blob_put_chunked(HttpRequest *req, HttpReply *rep, int fd);
  bool do_v1_blob_delete(
      HttpRequest *req, HttpReply *rep,
      const std::string &chunkid);
  bool do_v1_blob_get(HttpRequest *req, HttpReply *rep, bool body);
};


struct BlobServer {
  BlobStats stats;
  BlobConfig config;
  ThreadRunner threads;
  BlobService service;

  ~BlobServer();
  BlobServer(BlobServer &&o) = delete;
  BlobServer(const BlobServer &o) = delete;
  BlobServer();

  bool running() const;
  void configure(int fd);
};


enum class HttpRequestParsingStep {
  First = 0,
  HeaderName = 1,
  HeaderValue = 2,
  Body = 3,
};


struct HttpRequest {
  std::shared_ptr<BlobClient> client;
  int64_t when_active;
  int64_t when_headers;
  std::string url;
  std::map<std::string, std::string> headers;

  std::shared_ptr<opentracing::Tracer> tracer;

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
      std::shared_ptr<BlobClient> c,
      std::shared_ptr<opentracing::Tracer> t);

  bool consume_headers(int64_t dl);
  void add_body(std::vector<uint8_t> b, size_t offset);
  void save_header();
};


struct HttpReply {
  std::shared_ptr<BlobClient> client;
  HttpRequest *request;
  std::map<std::string, std::string> headers;

  ~HttpReply();
  HttpReply() = delete;
  HttpReply(const HttpReply &o) = delete;
  HttpReply(HttpReply &&o) = delete;
  explicit HttpReply(HttpRequest *req);

  bool write_error(int code);
  bool write_headers(int code, int64_t content_length);
  bool write_chunk(uint8_t *base, size_t length);
  bool write_chunk(char *base, size_t length);
  bool write_final_chunk();
};

#endif  // BLOB_SERVER_INTERNALS_HPP_
