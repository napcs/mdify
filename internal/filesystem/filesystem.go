package filesystem

import (
	"io"
	"os"
)

// FileSystem interface for abstracting file system operations
type FileSystem interface {
	Create(name string) (io.WriteCloser, error)
	MkdirAll(path string, perm int) error
	ReadFile(filename string) ([]byte, error)
	Stat(name string) (FileInfo, error)
}

// FileInfo interface for file information
type FileInfo interface {
	IsExist() bool
}

// OSFileSystem implements FileSystem using the actual OS
type OSFileSystem struct{}

func (fs OSFileSystem) Create(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

func (fs OSFileSystem) MkdirAll(path string, perm int) error {
	return os.MkdirAll(path, os.FileMode(perm))
}

func (fs OSFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (fs OSFileSystem) Stat(name string) (FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return &OSFileInfo{exists: false}, err
		}
		return nil, err
	}
	return &OSFileInfo{exists: true, info: info}, nil
}

// OSFileInfo implements FileInfo
type OSFileInfo struct {
	exists bool
	info   os.FileInfo
}

func (fi *OSFileInfo) IsExist() bool {
	return fi.exists
}