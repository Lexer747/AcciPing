// Use of this source code is governed by a GPL-2 license that can be found in the LICENSE file.
//
// Copyright 2024 Lexer747
//
// SPDX-License-Identifier: GPL-2.0-only

package files

import (
	"io"
	"os"

	"github.com/Lexer747/acci-ping/graph/data"
	"github.com/Lexer747/acci-ping/utils/check"
	"github.com/Lexer747/acci-ping/utils/errors"
)

// LoadFile will read a '.pings' file returning the data and the file handle (opened in read/write), or any
// error if a disk issue occurs or the data format was un-parsable.
func LoadFile(path string) (*data.Data, *os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0o777)
	if err != nil {
		return nil, nil, err
	}

	// File exists, read the data from it
	existingData := &data.Data{}
	fromFile, err := io.ReadAll(f)
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	if _, err = existingData.FromCompact(fromFile); err != nil {
		f.Close()
		return nil, nil, err
	}

	return existingData, f, nil
}

func MakeNewEmptyFile(path string, url string) (*data.Data, *os.File, error) {
	newFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o777)
	if err != nil {
		return nil, nil, err
	}
	d := data.NewData(url)
	// Write the initial data to the file on exit
	return d, newFile, d.AsCompact(newFile)
}

// LoadOrCreateFile will read a '.pings' file returning the data and the file handle (opened in read/write),
// or any error if a disk issue occurs or the data format was un-parsable. If the file isn't found at the
// given path then this specific error is swallowed and a new file is created with empty data pointing the
// given url.
func LoadOrCreateFile(path string, url string) (*data.Data, *os.File, error) {
	d, f, err := LoadFile(path)
	switch {
	case err != nil && !errors.Is(err, os.ErrNotExist):
		// Some error we are not expecting
		return nil, nil, err
	case err != nil && errors.Is(err, os.ErrNotExist):
		d, f, err = MakeNewEmptyFile(path, url)
		if err != nil {
			return nil, nil, err
		}
	}
	check.Check(d != nil && f != nil && d.URL == url, "data should be initialised")
	// Once the data is written/read reset the handle back to the start
	_, seekErr := f.Seek(0, 0)
	return d, f, errors.Join(seekErr, err)
}
