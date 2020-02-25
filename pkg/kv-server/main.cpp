//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include <unistd.h>
#include <signal.h>
#include <sys/stat.h>
#include <string>
#include <vector>
#include <iostream>
#include <sstream>
#include <thread>

#include <sys/sysinfo.h>

#include <glog/logging.h>
#include <gflags/gflags.h>
#include <rocksdb/db.h>
#include <nng/nng.h>
#include <nng/supplemental/util/platform.h>
#include <nng/protocol/reqrep0/req.h>
#include <nng/protocol/reqrep0/rep.h>

#include "protocol_generated.h"

using namespace ::kv::protocol;

/**
 * Specifies how many worker threads are parsing messages
 */
static unsigned long concurrency{1};

// ----------------------------------------------------------------------------

static std::string base_prefix(const flatbuffers::String *base) {
  std::stringstream ss;
  ss << base->str() << '/';
  return ss.str();
}

static std::string key_version(const flatbuffers::String *base,
    const flatbuffers::String *key, uint64_t version) {
  std::stringstream ss;
  ss << base->str() << '/' << key->str() << '/';
  ss.fill('0');
  ss << std::hex << version;
  return ss.str();
}

static std::string key_prefix(const flatbuffers::String *base,
    const flatbuffers::String *key) {
  std::stringstream ss;
  ss << base->str() << '/' << key->str() << '/';
  return ss.str();
}

static std::string key_last(const flatbuffers::String *base,
    const flatbuffers::String *key) {
  std::stringstream ss;
  ss << base->str() << '/' << key->str() << '/' << '~';
  return ss.str();
}

static void pack_message(nng_msg *mpu, flatbuffers::FlatBufferBuilder &fbb) {
  nng_msg_realloc(mpu, fbb.GetSize());
  memcpy(nng_msg_body(mpu), fbb.GetBufferPointer(), fbb.GetSize());
}

static void pack_error(nng_msg *mpu, int code, const char *message) {
  flatbuffers::FlatBufferBuilder fbb(256);
  auto why = fbb.CreateString(message);
  auto err = CreateErrorReply(fbb, code, why);
  auto msg = CreateMessage(fbb, 0, MessageKind_ErrorReply, err.Union());
  fbb.Finish(msg);
  LOG(INFO) << "ERROR " << code << " " << message;
  return pack_message(mpu, fbb);
}

enum class WorkerState {
  Read = 0,
  Write = 1
};

struct Worker {

  Worker(): state_{WorkerState::Read}, socket_{}, db_{nullptr} {}

  // AIO-based
  // Main callback registered in the AIO core
  void react();

  // AIO-based
  // Restart a request/reply cycle via the registration of a recv() command
  void rearm();

  // Synchronous
  void action();

  void handle_request(nng_msg *mpu);

  void handle_ping(const PingRequest *get, nng_msg *mpu);

  void handle_get(const GetRequest *get, nng_msg *mpu);

  void handle_put(const PutRequest *put, nng_msg *mpu);

  void handle_list(const ListRequest *list, nng_msg *mpu);

  // The last action issued by the current worker
  WorkerState state_;

  ::nng_socket socket_;
  ::nng_ctx ctx_;
  ::nng_aio *aio_;
  ::nng_msg *msg_;

  rocksdb::DB *db_;
};

void Worker::handle_request(nng_msg *request) {
  auto buf = static_cast<uint8_t *>(nng_msg_body(request));
  size_t len = nng_msg_len(request);

  // Check the format of the input
  ::flatbuffers::Verifier verifier(buf, len);
  if (!VerifyMessageBuffer(verifier))
    return pack_error(request, 400, "Malformed Request");

  // Call the handler if the message is accepted
  auto decoded = GetMessage(buf);
  switch (decoded->actual_type()) {
    case MessageKind_PingRequest:
      return handle_ping(decoded->actual_as_PingRequest(), request);
    case MessageKind_PutRequest:
      return handle_put(decoded->actual_as_PutRequest(), request);
    case MessageKind_GetRequest:
      return handle_get(decoded->actual_as_GetRequest(), request);
    case MessageKind_ListRequest:
      return handle_list(decoded->actual_as_ListRequest(), request);
    default:
      return pack_error(request, 418, "Unexpected request");
  }
}

class MsgAllocator : public flatbuffers::Allocator {
 private:
  uint8_t *base_;
  size_t size_;

 public:
  ~MsgAllocator() override {}

  MsgAllocator() = delete;
  MsgAllocator(MsgAllocator &&o) = delete;
  MsgAllocator(const MsgAllocator &o) = delete;

  MsgAllocator(::nng_msg *msg) {
    assert(msg != nullptr);
    base_ = (uint8_t*) ::nng_msg_body(msg);
    size_ = ::nng_msg_len(msg);
  }

  uint8_t* allocate (size_t size) override {
    if (size > size_) {
      LOG(ERROR) << "memory exhausted";
      return nullptr;
    }
    size_ -= size;
    uint8_t *out = base_ + size_;
    LOG(ERROR) << __func__ << '(' << size << ')' << " -> " << static_cast<void*>(out);
    return out;
  }

  void deallocate (uint8_t *p, size_t size) override {
    LOG(ERROR) << __func__ << '(' << static_cast<void*>(p) << ',' << size << ')';
    if (p == base_ + size_) {
      // Last block allocated
      size_ += size;
    } else {
      // Not the last block, we accept to lose it.
    }
  }

  uint8_t* reallocate_downward (
      uint8_t *old_p, size_t old_size, size_t new_size,
      size_t in_use_back, size_t in_use_front) override {
    LOG(ERROR) << __func__ << '(' << static_cast<void*>(old_p) << ',' << old_size << ',' << new_size << ',' << in_use_back << ',' << in_use_front << ')';
    assert(old_size < new_size);
    uint8_t *new_p{nullptr};
    if (old_p == base_ + size_) {
      // the block the last block allocated, we just stretch it
      LOG(ERROR) << "last";
      assert(size_ + old_size >= new_size);
      size_ = size_ + old_size - new_size;
      new_p = base_ + size_;
      memcpy_downward(old_p, old_size, new_p, new_size, in_use_back, in_use_front);
    } else {
      // Not the last block, we allocate a new block
      LOG(ERROR) << "non-last";
      new_p = allocate(new_size);
      memcpy_downward(old_p, old_size, new_p, new_size, in_use_back, in_use_front);
      deallocate(old_p, old_size);
    }
    LOG(ERROR) << __func__ << " -> " << static_cast<void*>(new_p);
    return new_p;
  }
};

void Worker::handle_ping(const PingRequest *ignored __attribute__ ((unused)),
    nng_msg *mpu) {

  ::nng_msg_realloc(mpu, 1024);
  MsgAllocator alloc(mpu);

  flatbuffers::FlatBufferBuilder fbb(0, &alloc);
  auto pongReply = CreatePingReply(fbb);
  auto msg = CreateMessage(fbb, 0, MessageKind_PingReply, pongReply.Union());
  fbb.Finish(msg);
  return pack_message(mpu, fbb);
}

void Worker::handle_get(const GetRequest *get, nng_msg *mpu) {
  const auto b0 = get->base();
  const auto k0 = get->key();
  const auto v0 = get->version();

  // Placeholders for the returned values
  rocksdb::ReadOptions roptions;
  roptions.fill_cache = true;
  int code{0};
  std::string v;

  if (b0->size() <= 0 || k0->size() <= 0) {
    code = 400;
  } else if (v0 == 0) {
    // The latest version is wanted
    auto prefix = key_prefix(b0, k0);
    auto last = key_last(b0, k0);

    auto it = db_->NewIterator(roptions);
    it->SeekForPrev(rocksdb::Slice(last.data(), last.size()));
    if (!it->Valid()) {
      code = 404;
    } else {
      if (it->key().starts_with(prefix)) {
        const auto itv = it->value();
        v.assign(itv.data(), itv.size());
      } else {
        code = 404;
      }
    }
    delete it;
  } else {
    // Perform the query on the specific version that is targeted
    auto k = key_version(b0, k0, v0);
    auto db_rc = db_->Get(roptions, rocksdb::Slice(k.data(), k.size()), &v);
    if (db_rc.IsNotFound())
      code = 404;
    else if (!db_rc.ok())
      code = 500;
  }

  if (code == 0) {
    flatbuffers::FlatBufferBuilder fbb(256);
    auto value = fbb.CreateString(v);
    auto getReply = CreateGetReply(fbb, value);
    auto msg = CreateMessage(fbb, 0, MessageKind_GetReply, getReply.Union());
    fbb.Finish(msg);
    return pack_message(mpu, fbb);
  } else {
    return pack_error(mpu, code, "argl");
  }
}

void Worker::handle_put(const PutRequest *put, nng_msg *mpu) {
  const auto b0 = put->base();
  const auto k0 = put->key();
  const auto d0 = put->value();
  const auto v0 = put->version();
  int code{0};

  if (b0->size() <= 0 || k0->size() <= 0 || d0->size() <= 0) {
    code = 400;
  } else {
    // Perform the query
    rocksdb::WriteOptions woptions;
    woptions.sync = false;
    auto key = key_version(b0, k0, v0);
    auto db_rc = db_->Put(woptions, rocksdb::Slice(key),
                          rocksdb::Slice(d0->data(), d0->size()));
    if (!db_rc.ok())
      code = 500;
  }

  // Pack the reply
  if (code == 0) {
    flatbuffers::FlatBufferBuilder fbb(256);
    auto putReply = CreatePutReply(fbb);
    auto msg = CreateMessage(fbb, 0, MessageKind_PutReply, putReply.Union());
    fbb.Finish(msg);
    return pack_message(mpu, fbb);
  } else {
    return pack_error(mpu, code, "argl");
  }
}

void Worker::handle_list(const ListRequest *list, nng_msg *mpu) {
  const auto b0 = list->base();
  const auto m0 = list->marker();
  const auto mv0 = list->markerVersion();
  auto max = list->max();

  int code{0};
  std::vector<std::pair<std::string, uint64_t >> result;

  if (max <= 0)
    max = 1000;

  if (b0->size() <= 0) {
    code = 400;
  } else {
    // Perform the query: generate an iterator
    rocksdb::ReadOptions roptions;
    roptions.fill_cache = true;
    auto it = db_->NewIterator(roptions);
    std::string prefix = base_prefix(b0);
    rocksdb::Slice sprefix(prefix);
    if (m0->size() == 0) {
      it->Seek(sprefix);
    } else if (mv0 <= 0) {
      // We have a marker but no version, we just skip to the first key that
      // does not match the prfix
      std::string marker = key_prefix(b0, m0);
      rocksdb::Slice smarker(marker);
      it->Seek(smarker);
      while (it->Valid() && it->key().starts_with(smarker))
        it->Next();
    } else {
      // We have a marker AND a version, so we just skip to the next context
      // that exactly match our marker
      std::string marker = key_version(b0, m0, mv0);
      rocksdb::Slice smarker(marker);
      it->Seek(smarker);
      while (it->Valid() && it->key() == smarker)
        it->Next();
    }

    // Unroll the iterator until the prefix is not matched
    for (uint16_t i = 0; i < max && it->Valid(); i++) {
      const auto k = it->key();
      if (!k.starts_with(sprefix))
        break;
      std::string key(k.data() + sprefix.size(), k.size() - sprefix.size());
      const auto slash = key.find('/', 1);
      if (slash != std::string::npos) {
        auto version = std::strtoull(key.c_str() + slash + 1, nullptr, 16);
        key.resize(slash);
        result.emplace_back(key, version);
      }
      it->Next();
    }
  }

  if (code == 0) {
    // Pack the reply
    flatbuffers::FlatBufferBuilder fbb(256);
    std::vector<flatbuffers::Offset<ListEntry>> entries;
    for (auto &item : result) {
      auto k = fbb.CreateString(item.first);
      auto e = CreateListEntry(fbb, k, item.second);
      entries.push_back(e);
    }
    auto ventries = fbb.CreateVector(entries);
    auto rep = CreateListReply(fbb, ventries);
    auto msg = CreateMessage(fbb, 0, MessageKind_ListReply, rep.Union());
    fbb.Finish(msg);
    return pack_message(mpu, fbb);
  } else {
    return pack_error(mpu, code, "argl");
  }
}

void Worker::action() {
  int rc;

  ::nng_msg *msg{nullptr};

  rc = ::nng_recvmsg(socket_, &msg, 0);
  assert(rc == 0);

  handle_request(msg);

  rc = ::nng_sendmsg(socket_, msg, 0);
  assert(rc == 0);
}

void Worker::react() {
  int rc;

  switch (state_) {

    case WorkerState::Read:
      rc = ::nng_aio_result(aio_);
      if (rc != 0)
        return rearm();
      msg_ = ::nng_aio_get_msg(aio_);
      assert(msg_ != nullptr);

      handle_request(msg_);

      ::nng_aio_set_msg(aio_, msg_);
      msg_ = nullptr;
      state_ = WorkerState::Write;
      return ::nng_ctx_send(ctx_, aio_);

    case WorkerState::Write:
      rc = ::nng_aio_result(aio_);
      if (rc != 0)
        LOG(INFO) << "write error: " << ::nng_strerror(rc);
      return rearm();
  }
}

void Worker::rearm() {
  state_ = WorkerState::Read;
  return ::nng_ctx_recv(ctx_, aio_);
}

static void worker_react(void *p) {
  assert(p != nullptr);
  static_cast<Worker*>(p)->react();
}

static int action(const char *url_front, ::rocksdb::DB *db) {
  std::vector<Worker> workers(concurrency);
  std::vector<std::thread> threads;

  ::nng_socket front;
  int rc = ::nng_rep0_open(&front);
  assert(rc == 0);
  //::nng_setopt_bool(front, NNG_OPT_TCP_KEEPALIVE, true);
  ::nng_setopt_bool(front, NNG_OPT_TCP_NODELAY, false);
  ::nng_setopt_size(front, NNG_OPT_RECVBUF, 4096 * 1024);
  ::nng_setopt_size(front, NNG_OPT_SENDBUF, 1024 * 1024);

  rc = ::nng_listen(front, url_front, nullptr, 0);
  assert(rc == 0);
  LOG(INFO) << "Socket ready at " << url_front;

  for (auto &worker : workers) {
    worker.db_ = db;
    worker.socket_ = front;
    rc = ::nng_aio_alloc(&worker.aio_, worker_react, &worker);
    assert(rc == 0);
    rc = ::nng_ctx_open(&worker.ctx_, worker.socket_);
    assert(rc == 0);
  }
  LOG(INFO) << "Workers ready";

  for (auto &worker : workers)
    threads.push_back(std::thread([&worker] {
      worker.rearm();
      for (;;) ::nng_msleep(5000);
    }));

  for (auto &thread : threads)
    thread.join();

  return 0;
}

static bool _check_basedir(const char *path) {
  struct ::stat st{};
  if (0 > stat(path, &st))
    return false;
  return S_ISDIR(st.st_mode);
}

static int action(const char *url_front, const char *db_path) {
  ::rocksdb::DB *db{nullptr};
  ::rocksdb::Options options{};
  options.create_if_missing = true;

  if (!_check_basedir(db_path)) {
    SYSLOG(ERROR) << "DB::Open(" << db_path << ") error: " << "not found";
    return -1;
  }

  auto dbrc = ::rocksdb::DB::Open(options, db_path, &db);
  if (!dbrc.ok()) {
    SYSLOG(ERROR) << "DB::Open(" << db_path << ") error: " << dbrc.ToString();
    return -1;
  }

  LOG(INFO) << "DB ready at " << db_path;
  int rc = action(url_front, db);
  delete db;
  return rc;
}

static void usage(const char *prog) {
  std::cout << "\nUSAGE:\n"
            << "  " << prog << " [-c INT] URL PATH\n"
            << "  " << prog << " -h\n"
            << "\nOPTIONS:\n"
            << "  -c INT : Set the number of workers (default: 1)\n"
            << "  -h     : Print this help section\n"
            << "  URL    : A connection string compatible with nanomsg-ng\n"
            << "  PATH   : The path to a directory with the necessary perms\n"
            << "\nEXAMPLE:\n"
            << "  " << prog << " tcp://127.0.0.1:6000 /var/lib/volume\n"
            << "  " << prog << " ipc:///var/run/kv.sock /var/lib/volume\n"
            << std::endl;
}

int main(int argc, char **argv) {
  FLAGS_logtostderr = true;
  google::InitGoogleLogging(argv[0]);

  concurrency = 1 + ::get_nprocs();

  int rc;
  while (-1 != (rc = ::getopt(argc, argv, "hc:"))) {
    switch (rc) {
      case 'h':
        usage(argv[0]);
        break;
      case 'c':
        concurrency = std::strtoul(optarg, nullptr, 0);
        break;
      case '?':
      default:
        ::exit(EXIT_FAILURE);
    }
  }

  if (concurrency <= 0) {
    LOG(ERROR) << "Not enough workers, " << concurrency << " specified";
    ::exit(EXIT_FAILURE);
  }

  if (optind + 1 >= argc) {
    usage(argv[0]);
    ::exit(EXIT_FAILURE);
  }

  rc = action(argv[optind], argv[1 + optind]);
  ::nng_fini();
  return rc;
}
