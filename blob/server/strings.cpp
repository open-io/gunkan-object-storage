//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "strings.hpp"

#include <cstdlib>
#include <cstring>

#include "charset.hpp"

#define CHOMP(s) while (s.back() == '/') { s.pop_back(); }

static CharSet _hexa("0123456789ABCDEFabcdef");

bool _starts_with(const std::string &s, const std::string &p) {
  return s.length() >= p.length() && !memcmp(s.data(), p.data(), p.length());
}

bool is_hexa(const std::string &s) {
  return _hexa.validate(s);
}

std::string path_parent(std::string path) {
  CHOMP(path);
  while (path.back() != '/')
    path.pop_back();
  CHOMP(path);
  return path;
}
