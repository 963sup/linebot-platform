package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// readPrivateBounded reads operator-selected local data with a fixed upper
// bound. The content is never included in errors or logs.
func readPrivateBounded(path string, limit int64) ([]byte, error) {
	if path == "" || limit <= 0 { return nil, errors.New("file and size limit are required") }
	file, err := os.Open(filepath.Clean(path))
	if err != nil { return nil, errors.New("cannot open operator file") }
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil { return nil, errors.New("cannot read operator file") }
	if int64(len(data)) > limit { return nil, errors.New("operator file exceeds size limit") }
	if len(data) == 0 { return nil, errors.New("operator file is empty") }
	return data, nil
}

// writePrivateFile must not overwrite or follow a symlink to a preexisting
// file. Token responses and downloaded media must never go to stdout.
func writePrivateFile(path string, content []byte) (err error) {
	if path == "" { return errors.New("output path is required") }
	f, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil { return errors.New("cannot create private output file") }
	defer func() {
		if e := f.Close(); err == nil && e != nil { err = errors.New("cannot close private output file") }
		if err != nil { _ = os.Remove(filepath.Clean(path)) }
	}()
	_, err = f.Write(content)
	if err != nil { return errors.New("cannot write private output file") }
	return nil
}
