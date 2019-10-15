//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt).  A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include <poll.h>
#include <getopt.h>
#include <signal.h>

#include <glog/logging.h>

#include <iostream>
#include <fstream>

#include "http.hpp"
#include "blob.hpp"
#include "threads.hpp"

#define CORO_STACK_SIZE 16384

#define COLD __attribute__((cold))


const int64_t one64{1};
bool flag_debug{true};
volatile bool flag_sys_running{true};

static int _make_server(std::string endpoint) COLD;
static void _usage(const BlobConfig &cfg) COLD;
static void _configure_main(const BlobConfig &config) COLD;
static void _sig_stop_react(int sig) COLD;
static void _sig_stop_rearm() COLD;
static void _parse_options(BlobConfig *cfg, int argc, char **argv) COLD;


int _make_server(std::string endpoint) {
  // Parse the endpoint into an address string
  auto colon = endpoint.rfind(':');
  if (colon == std::string::npos) {
    SYSLOG(ERROR) << "Invalid endpoint: " << "missing port";
    ::exit(EXIT_FAILURE);
  }

  auto str_port = endpoint.substr(colon + 1);
  endpoint.resize(colon);

  // Resolve the local address and establish a server on it
  ::dill_ipaddr addr{};
  int rc = ::dill_ipaddr_local(&addr,
      endpoint.c_str(), ::atoi(str_port.c_str()),
      DILL_IPADDR_PREF_IPV6);
  if (rc != 0) {
    SYSLOG(ERROR) << "Unresolvable endpoint: " << ::strerror(errno);
    ::exit(EXIT_FAILURE);
  }

  // Open a non-blocking socket
  const int srv_flags = SOCK_STREAM|SOCK_NONBLOCK|SOCK_CLOEXEC;
  int fd = ::socket(dill_ipaddr_family(&addr), srv_flags, 0);
  if (fd < 0) {
    SYSLOG(ERROR) << "Server socket failure: " << ::strerror(errno);
    ::exit(EXIT_FAILURE);
  }
  int opt{1};
  (void) ::setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

  // Bind it on the given endpoint
  rc = ::bind(fd, dill_ipaddr_sockaddr(&addr), dill_ipaddr_len(&addr));
  if (rc != 0) {
    SYSLOG(ERROR) << "Server bind failure: " << ::strerror(errno);
    ::exit(EXIT_FAILURE);
  }

  // Start listening on inbound connections
  rc = ::listen(fd, 256);
  if (rc != 0) {
    SYSLOG(ERROR) << "Server listen error: " << ::strerror(errno);
    ::exit(EXIT_FAILURE);
  }

  return fd;
}

void _usage(const BlobConfig &cfg) {
  std::cerr
    << "\n"
    << "Serve a repository of BLOB, exposed via the given network endpoints\n"
    << "and participating to the given Gunkan namespace\n"
    << "\n"
    << "Usage: " << cfg.argv0 << " [OPTIONS...] GUNKAN_NS ENDPOINT BASEDIR\n"
    << "\n"
    << " OPTIONS:" << "\n"
    << "  -h|--help               Display this help section\n"
    << "  -q|--quiet              Set the verbosity to 0\n"
    << "  -v|--verbose            Set to verbosity to a high level\n"
    << "  -d|--daemon             Daemonize the process\n"
    << "  -i|--init               Initiate the base directory\n"
    << "  -p|--pid     PATH       Specify the path of the pidfile\n"
    << "  --hash-width INT        Width of the levels of the FS hierarchy\n"
    << "  --hash-depth INT        Depth of the FS hierarchy\n"
    << "  --workers-ingress  INT  Max number of CNX being qualified\n"
    << "  --workers-be-read  INT  Max number of CNX in the Best-Effort pool\n"
    << "  --workers-be-write INT  Max number of CNX in the Best-Effort pool\n"
    << "  --workers-rt-read  INT  Max number of CNX in the Real-Time pool\n"
    << "  --workers-rt-write INT  Max number of CNX in the Real-Time pool\n"
    << std::endl;
}

void _parse_options(BlobConfig *cfg, int argc, char **argv) {
  static struct option options[] = {
    {"help", no_argument, 0, 'h'},
    {"quiet", no_argument, 0, 'q'},
    {"verbose", no_argument, 0, 'v'},
    {"daemon", no_argument, 0, 'd'},
    {"pid", required_argument, 0, 'p'},
    {"init", no_argument, 0, 'i'},
    {"hash-width", required_argument, 0, 'W'},
    {"hash-depth", required_argument, 0, 'D'},
    {"workers-ingress", required_argument, 0, 0},
    {"workers-be-read", required_argument, 0, 0},
    {"workers-be-write", required_argument, 0, 0},
    {"workers-rt-read", required_argument, 0, 0},
    {"workers-rt-write", required_argument, 0, 0},
    {nullptr, 0, nullptr, 0},
  };
  struct { const char *n; unsigned int *pv; } _long_only_opts[] = {
    {"hash-width", &cfg->hash_width},
    {"hash-depth", &cfg->hash_depth},
    {"workers-ingress", &cfg->workers_ingress},
    {"workers-be-read", &cfg->workers_be_read},
    {"workers-be-write", &cfg->workers_be_write},
    {"workers-rt-read", &cfg->workers_rt_read},
    {"workers-rt-write", &cfg->workers_rt_write},
  };

  for (;;) {
    int opt_index = 1;
    bool done{false};
    int c = ::getopt_long(argc, argv, "hvdip:W:D:", options, &opt_index);
    if (c < 0)
      break;
    switch (c) {
      case 'h':
        cfg->flag_help = true;
        break;
      case 'v':
        if (!cfg->flag_quiet)
          cfg->flag_verbose = true;
        break;
      case 'q':
        cfg->flag_verbose = false;
        cfg->flag_quiet = true;
        break;
      case 'd':
        cfg->flag_daemonize = true;
        break;
      case 'p':
        cfg->pidfile.assign(optarg);
        break;
      case 'i':
        cfg->flag_initiate = true;
        break;

      case 0:
        // Long options only
        done = false;
        for (auto o : _long_only_opts) {
          if (!strcmp(options[opt_index].name, o.n)) {
            *o.pv = ::atoi(optarg);
            done = true;
            break;
          }
        }
        if (!done) {
          std::cerr << "Unexpected option" << std::endl;
          ::exit(EXIT_FAILURE);
        }
        break;
      default:
        std::cerr << "Unexpected option" << std::endl;
        ::exit(EXIT_FAILURE);
    }
  }

  cfg->argv0.assign(argv[0]);

  if (optind + 2 >= argc) {
    _usage(*cfg);
    ::exit(EXIT_FAILURE);
  }

  cfg->nsname.assign(argv[optind]);
  cfg->endpoint.assign(argv[optind+1]);
  cfg->basedir.assign(argv[optind+2]);
}

void _configure_main(const BlobConfig &config) {
  if (config.flag_quiet) {
    if (!config.flag_daemonize) {
      stdout = ::freopen("/dev/null", "a", stdout);
      stderr = ::freopen("/dev/null", "a", stderr);
    }
  } else if (config.flag_verbose) {
    FLAGS_logtostderr = true;
    FLAGS_alsologtostderr = true;
  }

  google::InitGoogleLogging(config.argv0.c_str());

  if (config.flag_daemonize) {
    if (0 > ::daemon(1, 0)) {
      SYSLOG(ERROR) << "daemon() error: " << ::strerror(errno);
      ::exit(EXIT_FAILURE);
    }
  }

  if (!config.pidfile.empty()) {
    std::ofstream pid(config.pidfile, std::ios::binary);
    if (pid.bad() || pid.fail()) {
      SYSLOG(ERROR)
        << "pidfile error [" << config.pidfile << "]: "
        << ::strerror(errno);
      ::exit(EXIT_FAILURE);
    } else {
      pid << ::getpid();
    }
  }
}

static BlobServer server;
static ThreadRunner threads(&server.service);

void _sig_stop_react(int s __attribute__((unused))) {
  threads.stop();
  _sig_stop_rearm();
}

void _sig_stop_rearm() {
  signal(SIGINT,  _sig_stop_react);
  signal(SIGQUIT, _sig_stop_react);
  signal(SIGTERM, _sig_stop_react);
}

static void _poll_tokens() {
  static const char names[4][4] = { "ber", "bew", "rtr", "rtw", };
  struct pollfd pfdtab[4] = {
      {threads.executor_be_read.fd_tokens,  POLLIN, 0},
      {threads.executor_be_write.fd_tokens, POLLIN, 0},
      {threads.executor_rt_read.fd_tokens,  POLLIN, 0},
      {threads.executor_rt_write.fd_tokens, POLLIN, 0},
  };
  while (flag_sys_running) {
    // Reset the flags and wait for an event to happen
    for (auto &pfd : pfdtab)
      pfd.revents = 0;
    int rc = ::poll(pfdtab, 4, 5000);

    if (rc == 0)
      continue;
    if (rc < 0) {
      if (errno == EINTR)
        continue;
      break;
    }

    for (auto &pfd : pfdtab) {
      int64_t v{0};
      if (pfd.revents == 0)
        continue;
      ssize_t r = ::read(pfd.fd, &v, sizeof(v));
      if (r == sizeof(v) && v > 0) {
#if 0
        const char *name = names[i];
        LOG(INFO) << "TOKENS " << name << ' ' << v;
#endif
      }
    }
  }
}

int main(int argc, char **argv) {
  _parse_options(&server.config, argc, argv);
  _configure_main(server.config);
  _sig_stop_rearm();
  dill_stack_set_default_size(CORO_STACK_SIZE);
  dill_stack_set_cache_max(1024);

  if (!server.configure())
    ::exit(EXIT_FAILURE);
  threads.configure(_make_server(server.config.endpoint));
  threads.start();

  _poll_tokens();

  threads.stop();
  threads.join();
  if (!server.config.pidfile.empty())
    ::unlink(server.config.pidfile.c_str());
  return 0;
}
