//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "charset.hpp"

#include <sstream>

CharSet::~CharSet() {}

CharSet::CharSet() : allowed{} {}

CharSet::CharSet(const char *s) : CharSet() { set(s); }

bool CharSet::validate(const std::string &s) {
  for (auto c : s) {
    if (!allowed[c])
      return false;
  }
  return true;
}

void CharSet::set(const char *s) {
  for (; *s; ++s)
    allowed[*s] = true;
}
