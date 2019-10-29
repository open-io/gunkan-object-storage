/**
 * This file is part of oio-streams
 * Copyright (C) 2018-2019 OpenIO SAS
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 3.0 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library.
 */

#include "bytes.hpp"

#include <glog/logging.h>

#include <cassert>
#include <cstring>

static uint8_t empty[1] = {'\0'};
static auto end = make_static_slice(empty, 0);

Block::~Block() {}

AllocatedBlock* AllocatedBlock::Make(size_t n) {
  assert(n > 0);
  void *b{nullptr};
  int rc = ::posix_memalign(&b, 16, n);
  assert(rc == 0);
  return new AllocatedBlock(b, n);
}

AllocatedBlock::~AllocatedBlock() { free(base_); }

Slice make_allocated_slice(void *p, size_t s) {
  return Slice(std::shared_ptr<Block>(new AllocatedBlock(p, s)));
}

Slice make_static_slice(void *p, size_t s) {
  return Slice(std::shared_ptr<Block>(new StaticBlock(p, s)));
}

Slice make_string_slice(std::string s) {
  return Slice(std::shared_ptr<Block>(new StringBlock(s)));
}

Slice make_empty_slice() {
  return Slice(end);
}

Slice::~Slice() {}

Slice::Slice() : block_(end.block_), offset_{0}, size_{0} {}

Slice::Slice(const Slice &o)
    : block_(o.block_), offset_{o.offset_}, size_{o.size_} {
  assert(block_.get() != nullptr);
}

Slice::Slice(Slice &&o)
    : block_(o.block_), offset_{o.offset_}, size_{o.size_} {
  assert(block_.get() != nullptr);
}

Slice::Slice(std::shared_ptr<Block> b, size_t o, size_t s)
    : block_(b), offset_{o}, size_{s} {
  assert(block_.get() != nullptr);
  const size_t bs = b->size();
  assert(o <= bs);
  assert(s <= bs);
  assert(o + s <= bs);
}

Slice::Slice(std::shared_ptr<Block> b, size_t o)
    : block_(b), offset_{o}, size_{b->size() - o} {
  assert(block_.get() != nullptr);
  assert(o < b->size());
}

Slice::Slice(std::shared_ptr<Block> b)
    : block_(b), offset_{0}, size_{b->size()} {
  assert(block_.get() != nullptr);
}

void * Slice::data() const { return bytes(); }

uint8_t * Slice::bytes() const { return base() + offset_; }

char * Slice::c_str() const { return static_cast<char*>(data()); }

uint8_t *Slice::base() const {
  assert(block_.get() != nullptr);
  return static_cast<uint8_t *>(block_->data());
}

void Slice::trim(size_t s) {
  assert(block_.get() != nullptr);
  assert(s < size());
  size_ -= s;
}

void Slice::skip(size_t s) {
  assert(block_.get() != nullptr);
  assert(s <= size());
  size_ -= s;
  offset_ += s;
}

Slice Slice::sub(size_t offt) const {
  assert(block_.get() != nullptr);
  assert(offt < size());
  return Slice(block_, offset_ + offt, size_ - offt);
}

Slice Slice::sub(size_t offt, size_t s) const {
  assert(block_.get() != nullptr);
  assert(offt <= size());
  assert(s <= size());
  assert(offt + s <= size());
  return Slice(block_, offset_ + offt, s);
}

Slice Slice::sub(const void *p, size_t s) const {
  assert(p >= bytes());
  assert(p <= bytes() + size());
  return sub(static_cast<const uint8_t*>(p) - bytes(), s);
}

Slice& Slice::operator=(const Slice &o) {
  block_ = o.block_;
  offset_ = o.offset_;
  size_ = o.size_;
  return *this;
}

Slice concatenate(const std::vector<Slice> &blocks) {
  if (blocks.empty())
    return make_empty_slice();
  if (blocks.size() == 1)
    return blocks[0];

  size_t total = length(blocks);
  uint8_t *buffer = reinterpret_cast<uint8_t*>(valloc(total));
  total = 0;
  for (const auto &b : blocks) {
    memcpy(buffer + total, b.data(), b.size());
    total += b.size();
  }
  return make_allocated_slice(buffer, total);
}

size_t length(const std::vector<Slice> &blocks) {
  size_t total{0};
  for (const auto &b : blocks)
    total += b.size();
  return total;
}
