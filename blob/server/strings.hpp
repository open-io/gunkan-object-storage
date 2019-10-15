//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_STRINGS_HPP_
#define BLOB_SERVER_STRINGS_HPP_

#include <sys/types.h>

#include <cstdint>
#include <string>

#define ALIGN 8

static char CRLF[] = "\r\n";

static inline uint8_t * _align(uint8_t *buf) __attribute__((hot, pure, const));

static inline uint8_t * _align(uint8_t *buf) {
  return buf + ALIGN - (reinterpret_cast<ulong>(buf) % ALIGN);
}

bool _starts_with(const std::string &s, const std::string &p);

bool is_hexa(const std::string &s);

std::string path_parent(std::string path);

#endif  // BLOB_SERVER_STRINGS_HPP_
