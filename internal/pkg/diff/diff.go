// Package diff implements diffing.
//
// Should primarily be used for testing.
//
// Copied from https://github.com/golang/go/blob/master/src/cmd/gofmt/gofmt.go
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// https://github.com/golang/go/blob/master/LICENSE
package diff

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Diff does a diff.
func Diff(
	b1 []byte,
	b2 []byte,
	filename1 string,
	filename2 string,
	keepTimestamps bool,
) ([]byte, error) {
	f1, err := writeTempFile("", "", b1)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Remove(f1)
	}()

	f2, err := writeTempFile("", "", b2)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.Remove(f2)
	}()

	cmd := "diff"
	if runtime.GOOS == "plan9" {
		cmd = "/bin/ape/diff"
	}

	data, err := exec.Command(cmd, "-u", f1, f2).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		return tryModifyHeader(data, filename1, filename2, keepTimestamps), nil
	}
	return nil, err
}

func writeTempFile(dir, prefix string, data []byte) (string, error) {
	file, err := ioutil.TempFile(dir, prefix)
	if err != nil {
		return "", err
	}
	_, err = file.Write(data)
	if err1 := file.Close(); err == nil {
		err = err1
	}
	if err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func tryModifyHeader(
	diff []byte,
	filename1 string,
	filename2 string,
	keepTimestamps bool,
) []byte {
	bs := bytes.SplitN(diff, []byte{'\n'}, 3)
	if len(bs) < 3 {
		return diff
	}
	// Preserve timestamps.
	var t0, t1 []byte
	if keepTimestamps {
		if i := bytes.LastIndexByte(bs[0], '\t'); i != -1 {
			t0 = bs[0][i:]
		}
		if i := bytes.LastIndexByte(bs[1], '\t'); i != -1 {
			t1 = bs[1][i:]
		}
	}
	// Always print filepath with slash separator.
	filename1 = filepath.ToSlash(filename1)
	filename2 = filepath.ToSlash(filename2)
	if filename1 == filename2 {
		filename1 = filename1 + ".orig"
	}
	bs[0] = []byte(fmt.Sprintf("--- %s%s", filename1, t0))
	bs[1] = []byte(fmt.Sprintf("+++ %s%s", filename2, t1))
	return bytes.Join(bs, []byte{'\n'})
}
