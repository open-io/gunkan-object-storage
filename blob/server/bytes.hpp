//
// Copyright 2019 OpenIO SAS
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

#ifndef BLOB_SERVER_BYTES_HPP_
#define BLOB_SERVER_BYTES_HPP_

#include <string>
#include <memory>
#include <vector>

class Slice;

/**
 * A block is an abstraction of a data segment that manages the lifecycle
 * and the memory management in multi-threaded applications.
 *
 * Only two accessors: the first to the base pointer of the buffer and the
 * second to the length of the buffer.
 *
 * The common usage of a block is to created a slice.
 */
class Block {
 public:
  virtual ~Block();

  virtual void *data() const = 0;

  virtual size_t size() const = 0;
};

/**
 * A StaticBlock is a Block that wraps a buffer that won't be freed when the
 * StaticBlock will be destroyed.
 */
class StaticBlock : public Block {
 public:
  ~StaticBlock() override {}
  StaticBlock() = delete;
  StaticBlock(StaticBlock &&o) = delete;
  StaticBlock(const StaticBlock &o) = delete;
  StaticBlock(void *b, size_t s) : Block(), base_{b}, size_{s} {}

  void *data() const override { return base_; }

  size_t size() const override { return size_; }

 private:
  void *base_;
  size_t size_;
};

/**
 * Call free() on the undrlying block when the destructor is called.
 */
class AllocatedBlock : public Block {
 public:
  ~AllocatedBlock() override;
  AllocatedBlock() = delete;
  AllocatedBlock(AllocatedBlock &&o) = delete;
  AllocatedBlock(const AllocatedBlock &o) = delete;

  AllocatedBlock(void *b, size_t s) : Block(), base_{b}, size_{s} {}

  void *data() const override { return base_; }

  size_t size() const override { return size_; }

  static AllocatedBlock* Make(size_t n);
 private:
  void *base_;
  size_t size_;
};

/**
 *
 */
class CustomBlock : public Block {
 public:
  using Hook = void (*)(void *);

  ~CustomBlock() override { clean_(base_); }
  CustomBlock() = delete;
  CustomBlock(CustomBlock &&o) = delete;
  CustomBlock(const CustomBlock &o) = delete;

  CustomBlock(void *b, size_t s, Hook h)
      : Block(), base_{b}, size_{s}, clean_{h} {}

  void *data() const override { return base_; }

  size_t size() const override { return size_; }

 private:
  void *base_;
  size_t size_;
  Hook clean_;
};

/**
 * Build a block arround a copy of a string.
 */
class StringBlock : public Block {
 public:
  ~StringBlock() override {}
  StringBlock() = delete;
  StringBlock(StringBlock &&o) = delete;
  StringBlock(const StringBlock &o) = delete;

  explicit StringBlock(std::string s): base_{s} {}

  void *data() const override {
    return (void*) base_.data();  // NOLINT
  }

  size_t size() const override { return base_.size(); }

 private:
  std::string base_;
};

template <int N>
class ArrayBlock : public Block {
 public:
  ~ArrayBlock() override {}
  ArrayBlock() : base_() {}
  ArrayBlock(ArrayBlock &&o) = delete;
  ArrayBlock(const ArrayBlock &o) = delete;

  void *data() const override {
    return (void*) base_.data();  // NOLINT
  }

  size_t size() const override { return base_.size(); }

 private:
  std::array<uint8_t, N> base_;
};

/**
 * A view on a block.
 * Slices are built of of blocks to manage reference counts.
 */
class Slice {
 public:
  /**
   * Destroy the current Slice.
   */
  ~Slice();

  /**
   * Forbid the creation of any empty Slice
   */
  Slice();

  /**
   * Copy the given Slice
   * @param o a reference to the Slice to be copied
   */
  Slice(const Slice &o);

  /**
   * Move the given Slice
   * @param o the reference of the Slice to be moved
   */
  Slice(Slice &&o);

  /**
   * Build a view on the whole block
   * @param b
   */
  explicit Slice(std::shared_ptr<Block> b);

  /**
   * Build a view on the tail of the block starting at <o>.
   * @param b
   * @param o
   */
  Slice(std::shared_ptr<Block> b, size_t o);

  /**
   * Build a view on the portion of the block <b> starting at its offset
   * <o> and as long as <s>.
   * @param b
   * @param o
   * @param s
   */
  Slice(std::shared_ptr<Block> b, size_t o, size_t s);

  /**
   * Returns an abstract pointer to the buffer
   * @return the address of the buffer
   */
  void *data() const;

  /**
   * Returns a C'string-typed pointer to the buffer.
   * There is no guarantee the buffer is '\0' terminated.
   * @return the address of the buffer
   */
  char *c_str() const;

  /**
   * Returns the pointer to the buffer, typed as a byte array.
   * @return the address of the buffer
   */
  uint8_t *bytes() const;

  /**
   * Returns the size of the underlying buffer
   * @return the size of the buffer.
   */
  size_t size() const { return size_; }

  /**
   * Returns a new Slice of the same buffer starting at offset 'offt'
   * @param offt a valid offset relative to the current buffer address
   * @return a valid Slice
   */
  Slice sub(size_t offt) const;

  /**
   *
   * @param o
   * @param s
   * @return
   */
  Slice sub(size_t o, size_t s) const;

  /**
   * Build a slice pointing to the block of the current slice, based on an
   * pointer and a size. The pointer `p` MUST be in the current slice.
   * @param p
   * @param s
   * @return
   */
  Slice sub(const void *p, size_t s) const;

  /**
   * Skip <s> bytes at the beginning of the slice.
   * @param s
   */
  void skip(size_t s);

  /**
   * Remove <s> bytes at the end of the slice.
   * @param s
   */
  void trim(size_t s);

  Slice& operator=(const Slice &o);

 protected:
  /**
   * Returns the base of the underlying Block, regardless of the configured
   * offset.
   * @return
   */
  uint8_t *base() const;

 private:
  std::shared_ptr<Block> block_;
  size_t offset_;
  size_t size_;
};

Slice concatenate(const std::vector<Slice> &blocks);

size_t length(const std::vector<Slice> &blocks);

Slice make_static_slice(void *p, size_t s);

Slice make_allocated_slice(void *p, size_t s);

Slice make_string_slice(std::string s);

Slice make_empty_slice();

#endif  // BLOB_SERVER_BYTES_HPP_
