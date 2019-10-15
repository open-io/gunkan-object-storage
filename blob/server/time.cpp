//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "time.hpp"

#include <sys/time.h>

#include <ctime>

uint64_t monotonic_microseconds() {
  struct timespec ts{};
  (void) clock_gettime(CLOCK_MONOTONIC, &ts);
  return (ts.tv_nsec / 1000) + (ts.tv_sec * 1000000);
}
