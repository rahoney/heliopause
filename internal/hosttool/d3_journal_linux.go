//go:build linux

package hosttool

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const (
	d3Directory    = "/var/lib/heliopause-recovery/journal"
	d3JournalLimit = 1 << 20
)

type d3Record struct {
	Owner           string            `json:"owner"`
	Mutations       uint32            `json:"mutations"`
	Containers      []d3Container     `json:"containers,omitempty"`
	Keepers         []string          `json:"keepers,omitempty"`
	Policies        []d3PolicyRecord  `json:"policies,omitempty"`
	Observer        *d3ObserverRecord `json:"observer,omitempty"`
	ObserverPending bool              `json:"observer_pending,omitempty"`
}

type d3Container struct {
	ID    string `json:"id"`
	Scope string `json:"scope"`
}

type d3Journal struct {
	directory string
	lock      *os.File
	record    *d3Record
	sockets   []d3Endpoint
}

func openD3Journal(directory string, initialize bool) (*d3Journal, error) {
	parent := filepath.Dir(directory)
	fresh := false
	if initialize {
		err := os.Mkdir(directory, 0o700)
		fresh = err == nil
		if !fresh && !errors.Is(err, os.ErrExist) {
			return nil, err
		}
	}
	if err := d3CheckFile(directory, true); err != nil {
		return nil, err
	}
	if fresh {
		if err := d3SyncDirectory(parent); err != nil {
			return nil, err
		}
	}
	lock, err := os.OpenFile(filepath.Join(directory, "lock"), os.O_CREATE|os.O_RDWR|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	j := &d3Journal{directory: directory, lock: lock}
	if err := d3CheckFile(lock.Name(), false); err != nil {
		_ = lock.Close()
		return nil, err
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = lock.Close()
		return nil, errors.New("another recovery helper owns the journal")
	}
	if fresh {
		err = j.write(nil)
	} else {
		err = j.read()
	}
	if err != nil {
		_ = lock.Close()
		return nil, err
	}
	return j, nil
}

func d3CheckFile(path string, directory bool) error {
	var s unix.Stat_t
	if err := unix.Lstat(path, &s); err != nil {
		return err
	}
	mode := uint32(unix.S_IFREG | 0o600)
	if directory {
		mode = unix.S_IFDIR | 0o700
	}
	if s.Mode != mode || s.Uid != uint32(os.Geteuid()) || (!directory && s.Nlink != 1) {
		return errors.New("recovery journal ownership or mode is invalid")
	}
	return nil
}

func (j *d3Journal) read() error {
	path := filepath.Join(j.directory, "state.json")
	if err := d3CheckFile(path, false); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, d3JournalLimit))
	decoder.DisallowUnknownFields()
	var disk struct {
		Version int          `json:"version"`
		Record  *d3Record    `json:"record"`
		Sockets []d3Endpoint `json:"sockets,omitempty"`
	}
	info, err := file.Stat()
	if err != nil || info.Size() >= d3JournalLimit || decoder.Decode(&disk) != nil || decoder.Decode(&struct{}{}) != io.EOF || disk.Version != 1 ||
		disk.Record != nil && !validProcessOwner(disk.Record.Owner) {
		return errors.New("recovery journal is invalid")
	}
	j.record = disk.Record
	j.sockets = disk.Sockets
	return nil
}

// write completes both syncs before reporting that an instruction was issued.
// The protected directory and its process lock serialize the file replacement.
func (j *d3Journal) write(record *d3Record) error {
	body, err := json.Marshal(struct {
		Version int          `json:"version"`
		Record  *d3Record    `json:"record"`
		Sockets []d3Endpoint `json:"sockets,omitempty"`
	}{1, record, j.sockets})
	if err != nil || len(body) >= d3JournalLimit {
		return errors.New("recovery journal exceeds its bound")
	}
	file, err := os.CreateTemp(j.directory, ".recovery-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	n, writeErr := file.Write(body)
	if n != len(body) {
		writeErr = errors.Join(writeErr, io.ErrShortWrite)
	}
	if err := errors.Join(writeErr, file.Sync(), file.Close()); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), filepath.Join(j.directory, "state.json")); err != nil {
		return err
	}
	if err := d3SyncDirectory(j.directory); err != nil {
		return err
	}
	j.record = record
	return nil
}

func d3SyncDirectory(path string) error {
	dir, err := os.OpenFile(path, os.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
