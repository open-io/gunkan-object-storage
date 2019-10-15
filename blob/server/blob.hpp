//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_BLOB_HPP_
#define BLOB_SERVER_BLOB_HPP_

#include <atomic>
#include <memory>
#include <string>

#include "io.hpp"
#include "http.hpp"
#include "blobid.hpp"


struct BlobConfig;
struct BlobStats;
class BlobService;
struct BlobServer;


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

  bool flag_fallocate;
  bool flag_fadvise_upload;
  bool flag_fadvise_download;
  bool flag_readahead;
  bool flag_fsync_data;
  bool flag_fsync_dir;

  // Configuration directives with an impact on the main routine
  std::string argv0;
  std::string endpoint;
  std::string nsname;
  std::string pidfile;
  std::string basedir;

  ~BlobConfig() {}

  BlobConfig(const BlobConfig &o) = delete;

  BlobConfig(BlobConfig &&o) = delete;

  BlobConfig() : hash_width{3}, hash_depth{1},
                 workers_ingress{64},
                 workers_be_read{1024}, workers_be_write{1024},
                 workers_rt_read{8}, workers_rt_write{8},
                 flag_help{false}, flag_verbose{false},
                 flag_daemonize{false}, flag_initiate{false},
                 argv0(), endpoint(), nsname(), pidfile(), basedir() {}
};


struct BlobStats {
  // Bytes (counters)
  std::atomic<uint64_t> b_in;
  std::atomic<uint64_t> b_out;

  // Time (counters)
  std::atomic<uint64_t> t_info;
  std::atomic<uint64_t> t_status;
  std::atomic<uint64_t> t_put;
  std::atomic<uint64_t> t_get;
  std::atomic<uint64_t> t_head;
  std::atomic<uint64_t> t_delete;
  std::atomic<uint64_t> t_list;
  std::atomic<uint64_t> t_other;

  // Hits (counters)
  std::atomic<uint64_t> h_info;
  std::atomic<uint64_t> h_status;
  std::atomic<uint64_t> h_put;
  std::atomic<uint64_t> h_get;
  std::atomic<uint64_t> h_head;
  std::atomic<uint64_t> h_delete;
  std::atomic<uint64_t> h_list;
  std::atomic<uint64_t> h_other;

  // Response codes (counters)
  std::atomic<uint64_t> c_200;
  std::atomic<uint64_t> c_201;
  std::atomic<uint64_t> c_204;
  std::atomic<uint64_t> c_206;
  std::atomic<uint64_t> c_400;
  std::atomic<uint64_t> c_403;
  std::atomic<uint64_t> c_404;
  std::atomic<uint64_t> c_405;
  std::atomic<uint64_t> c_408;
  std::atomic<uint64_t> c_409;
  std::atomic<uint64_t> c_418;
  std::atomic<uint64_t> c_499;
  std::atomic<uint64_t> c_502;
  std::atomic<uint64_t> c_503;
  std::atomic<uint64_t> c_50X;
};


class BlobService : public HttpHandler {
 private:
  BlobServer *server;
  int fd_basedir;

 public:
  ~BlobService();

  BlobService() = delete;

  BlobService(BlobService &&o) = delete;

  BlobService(const BlobService &o) = delete;

  explicit BlobService(BlobServer *s);

  bool configure(std::string basedir);

  /**
   * Build the path of a blob for the given ID, relative to the base directory.
   * @param id
   * @return
   */
  std::string fullpath(const BlobId &id);

  void execute(
      std::unique_ptr<HttpRequest> req,
      std::unique_ptr<HttpReply> rep) override;

  /**
   * Describe the current service
   * @param req
   * @param rep
   */
  void do_info(
      HttpRequest *req,
      HttpReply *rep);

  /**
   * Reply the usage statistics of the service
   * @param req
   * @param rep
   */
  void do_v1_status(
      HttpRequest *req,
      HttpReply *rep);

  /**
   * Handle to requests targeting path prefixed with '/blob'
   * @param req
   * @param rep
   */
  void do_v1_blob(
      HttpRequest *req,
      HttpReply *rep);

  /**
   *
   * @param req
   * @param rep
   */
  void do_v1_list(
      HttpRequest *req,
      HttpReply *rep);

 private:
  /**
   * Handles PUT requests with a '/blob' prefix (i.e. BLOB uploads)
   * @param req
   * @param rep
   * @param path
   */
  void do_v1_blob_put(
      HttpRequest *req,
      HttpReply *rep,
      const std::string &path);

  void do_v1_blob_delete(
      HttpRequest *req,
      HttpReply *rep,
      const std::string &path);

  void do_v1_blob_get(
      HttpRequest *req,
      HttpReply *rep,
      const std::string &path, bool body);

 private:
  int _upload_body_inline(
      HttpRequest *req,
      FileAppender *chunk);

  int _upload_body_chunked(
      HttpRequest *req,
      FileAppender *chunk);

  void _upload_finish(const FileAppender &chunk);
};


struct BlobServer {
  BlobStats stats;
  BlobConfig config;
  BlobService service;

  ~BlobServer();

  BlobServer(BlobServer &&o) = delete;

  BlobServer(const BlobServer &o) = delete;

  BlobServer();

  bool configure();
};


#endif  // BLOB_SERVER_BLOB_HPP_
