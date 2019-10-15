//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_CHARSET_HPP_
#define BLOB_SERVER_CHARSET_HPP_

#include <array>
#include <string>


class CharSet {
 private:
  std::array<bool, 256> allowed;
 public:
  ~CharSet();

  CharSet();

  explicit CharSet(const char *s);

  void set(const char *s);

  bool validate(const std::string &s);
};


#endif  // BLOB_SERVER_CHARSET_HPP_
