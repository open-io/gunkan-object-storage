//
// Copyright 2019 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_BLOBID_HPP_
#define BLOB_SERVER_BLOBID_HPP_

#include <string>


struct BlobId {
  std::string id_content;
  std::string id_part;
  unsigned int position;

  std::string encode() const;

  bool decode(const std::string &s);

  static bool decode(BlobId *id, const std::string &s) {
    return id->decode(s);
  }
};


#endif  // BLOB_SERVER_BLOBID_HPP_
