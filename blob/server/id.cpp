//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#include "internals.hpp"

std::string BlobId::encode() const {
  std::stringstream ss;
  ss << id_content << ',' << id_part << ',' << position;
  return ss.str();
}

enum class BlobParser { Content, Part, Position };

bool BlobId::decode(const std::string &s) {
  std::stringstream ss_content, ss_part, ss_position;

  id_content.clear();
  id_part.clear();
  position = 0;

  BlobParser parser{BlobParser::Content};
  for (auto c : s) {
    switch (parser) {
      case BlobParser::Content:
        if (c == ',') {
          parser = BlobParser::Part;
        } else {
          ss_content << c;
        }
        break;
      case BlobParser::Part:
        if (c == ',') {
          parser = BlobParser::Position;
        } else {
          ss_part << c;
        }
        break;
      case BlobParser::Position:
        ss_position << c;
        break;
    }
  }

  if (parser != BlobParser::Position)
    return false;
  auto spos = ss_position.str();
  if (spos.size() == 0)
    return false;

  id_content.assign(ss_content.str());
  id_part.assign(ss_part.str());
  position = std::atoi(spos.c_str());
  return is_hexa(id_content) && is_hexa(id_part);
}

