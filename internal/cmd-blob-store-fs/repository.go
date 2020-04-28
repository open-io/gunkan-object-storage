//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package cmd_blob_store_fs

import (
	"fmt"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"golang.org/x/sys/unix"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Repo interface {
	Create(id gunkan.BlobId) (BlobBuilder, error)
	Open(blobId string) (BlobReader, error)
	Delete(blobId string) error
}

type BlobReader interface {
	Stream() *os.File
	Close()
}

type BlobBuilder interface {
	Stream() *os.File
	Commit() (string, error)
	Abort() error
}

type fsPostRepo struct {
	fdBase   int
	pathBase string

	rwRand sync.RWMutex
	idRand *rand.Rand

	// Control the way a filename is hashed to get the directory hierarchy
	hashWidth uint

	// Control the guarantees given before replying to the client
	syncFile bool
	syncDir  bool
}

type fsPostRW struct {
	file *os.File
	repo *fsPostRepo
	id   gunkan.BlobId
}

type fsPostRO struct {
	file *os.File
	repo *fsPostRepo
}

func MakePostNamed(basedir string) (Repo, error) {
	var err error
	r := fsPostRepo{
		fdBase:    -1,
		pathBase:  basedir,
		hashWidth: 4,
		syncFile:  false,
		syncDir:   false}

	r.fdBase, err = syscall.Open(r.pathBase, flagsOpenDir, 0)
	if err != nil {
		return nil, err
	}

	r.idRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	return &r, nil
}

func (r *fsPostRepo) relpath(objname string) (string, error) {
	sb := strings.Builder{}
	sb.Grow(16)
	if r.hashWidth > 0 {
		sb.WriteString(objname[0:r.hashWidth])
		sb.WriteRune('/')
	}
	sb.WriteString(objname[r.hashWidth:])
	return sb.String(), nil
}

func (r *fsPostRepo) mkdir(path string, retry bool) error {
	err := unix.Mkdirat(r.fdBase, path, 0755)
	if err == nil || os.IsExist(err) {
		return nil
	}
	if os.IsNotExist(err) {
		if err = r.mkdir(filepath.Dir(path), true); err == nil {
			return r.mkdir(path, false)
		}
	}
	return err
}

func (r *fsPostRepo) createOrRetry(path string, retry bool) (*os.File, error) {
	fd, err := unix.Openat(r.fdBase, path, flagsCreate, 0644)
	if err != nil {
		if retry && os.IsNotExist(err) {
			err = r.mkdir(filepath.Dir(path), true)
			if err == nil {
				return r.createOrRetry(path, false)
			}
		}
		return nil, err
	}

	return os.NewFile(uintptr(fd), path), nil
}

func (r *fsPostRepo) Delete(relpath string) error {
	return unix.Unlinkat(r.fdBase, relpath, 0)
}

func (r *fsPostRepo) nextId() string {
	d := (time.Now().UnixNano() / (1024 * 1024 * 256)) % 65536
	f := uint32(r.idRand.Int31n(1024 * 1024))
	return fmt.Sprintf("%04X%05X", d, f)
}

func (r *fsPostRepo) Create(id gunkan.BlobId) (BlobBuilder, error) {
	cid := r.nextId()

	pathFinal, err := r.relpath(cid)
	if err != nil {
		return nil, err
	}

	var f *os.File
	f, err = r.createOrRetry(pathFinal, true)
	return &fsPostRW{file: f, repo: r, id: id}, err
}

func (r *fsPostRepo) Open(realid string) (BlobReader, error) {
	var err error
	relpath, err := r.relpath(realid)
	if err != nil {
		return nil, err
	}

	var fd int
	fd, err = unix.Openat(r.fdBase, relpath, flagsOpenRead, 0)
	if err != nil {
		return nil, err
	}

	return &fsPostRO{file: os.NewFile(uintptr(fd), relpath), repo: r}, nil
}

func (f *fsPostRW) Stream() *os.File {
	return f.file
}

func (f *fsPostRW) Abort() error {
	if f == nil || f.file == nil {
		return nil
	}
	err := unix.Unlinkat(f.repo.fdBase, f.file.Name(), 0)
	_ = f.file.Close()
	return err
}

func (f *fsPostRW) Commit() (string, error) {
	if f == nil || f.file == nil {
		panic("Invalid file being commited")
	}

	var err error
	if f.repo.syncFile {
		err = f.file.Sync()
	}
	if f.repo.syncDir {
		err = unix.Fdatasync(int(f.file.Fd()))
	}

	_ = f.file.Close()
	return f.file.Name(), err
}

func (f *fsPostRO) Stream() *os.File {
	return f.file
}

func (f *fsPostRO) Close() {
	_ = f.file.Close()
}
