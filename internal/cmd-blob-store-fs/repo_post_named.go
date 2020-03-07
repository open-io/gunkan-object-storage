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
	"errors"
	"github.com/jfsmig/object-storage/pkg/gunkan"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type fsPostRepo struct {
	fdBase   int
	pathBase string

	// Control the way a filename is hashed to get the directory hierarchy
	hashWidth uint
	hashDepth uint

	sync     bool
	datasync bool
}

type fsPostRW struct {
	file      *os.File
	repo      *fsPostRepo
	pathFinal string
}

type fsPostRO struct {
	file *os.File
	repo *fsPostRepo
}

func MakePostNamed(basedir string) (Repo, error) {
	var err error
	r := fsPostRepo{pathBase: basedir, hashWidth: 3, hashDepth: 1, sync: false, datasync: false}
	r.fdBase, err = syscall.Open(r.pathBase, flagsOpenDir, 0)
	if err != nil {
		return nil, err
	} else {
		return &r, nil
	}
}

func (r *fsPostRepo) relpath(objname string) (string, error) {
	sb := strings.Builder{}
	sb.Grow(256)
	any := false
	for i := uint(0); i < r.hashDepth; i++ {
		if any {
			sb.WriteRune('/')
		}
		any = true
		start := i * r.hashWidth
		sb.WriteString(objname[start : start+r.hashWidth])
	}
	if any {
		sb.WriteRune('/')
	}
	sb.WriteString(objname[r.hashWidth*r.hashDepth:])
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

func (r *fsPostRepo) Create(obj gunkan.BlobId) (BlobBuilder, error) {
	encoded := obj.Encode()
	pathFinal, err := r.relpath(encoded)
	if err != nil {
		return nil, err
	}

	pathTemp := strings.Replace(pathFinal, ",", "@", 1)
	if pathTemp == pathFinal {
		return nil, errors.New("Malformed blob path")
	}

	f, err := r.createOrRetry(pathFinal, true)
	return &fsPostRW{
		file:      f,
		pathFinal: pathFinal,
		repo:      r}, nil
}

func (r *fsPostRepo) Open(blobid string) (BlobReader, error) {
	var err error
	relpath, err := r.relpath(blobid)
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
	if f.file == nil {
		panic("Invalid file being commited")
	}

	var err error
	if f.repo.sync {
		err = f.file.Sync()
	} else if f.repo.datasync {
		err = unix.Fdatasync(int(f.file.Fd()))
	}
	err = unix.Renameat(f.repo.fdBase, f.file.Name(), f.repo.fdBase, f.pathFinal)
	_ = f.file.Close()
	return f.pathFinal, err
}

func (f *fsPostRO) Stream() *os.File {
	return f.file
}

func (f *fsPostRO) Close() {
	_ = f.file.Close()
}
