// Copyright 2012 Jonas mg
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fileutil

import (
	"io"
	"io/ioutil"
	"os"
)

// Copy copies file from 'source' to file in 'dest' preserving the mode attributes.
// It makes a backup if the global variable 'DoBackup' is set to true.
func Copy(source, dest string) (err error) {
	// Don't backup files of backup.
	if DoBackup && dest[len(dest)-1] != '~' {
		if err = Backup(dest); err != nil {
			return
		}
	}

	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		if err2 := srcFile.Close(); err2 != nil && err == nil {
			err = err2
		}
	}()

	srcInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return err
	}

	_, err = io.Copy(dstFile, srcFile)
	err2 := dstFile.Close()
	if err2 != nil && err == nil {
		err = err2
	}
	if err != nil {
		return err
	}

	Log.Printf("File %q copied at %q", source, dest)
	return nil
}

// Create creates a new file with b bytes.
func Create(filename string, b []byte) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	_, err = file.Write(b)
	err2 := file.Close()
	if err2 != nil && err == nil {
		err = err2
	}
	if err != nil {
		return err
	}

	Log.Printf("File %q created", filename)
	return nil
}

// CreateString is like Create, but writes the contents of string s rather than
// an array of bytes.
func CreateString(filename, s string) error {
	return Create(filename, []byte(s))
}

// Overwrite truncates the named file to zero and writes len(b) bytes.
// It makes a backup if the global variable 'DoBackup' is set to true.
// It returns an error, if any.
func Overwrite(filename string, b []byte) (err error) {
	if DoBackup {
		if err = Backup(filename); err != nil {
			return err
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	_, err = file.Write(b)
	err2 := file.Close()
	if err2 != nil && err == nil {
		err = err2
	}
	if err != nil {
		return err
	}

	Log.Printf("File %q overwritted", filename)
	return nil
}

// OverwriteString is like Overwrite, but writes the contents of string s rather
// than an array of bytes.
func OverwriteString(filename, s string) error {
	return Overwrite(filename, []byte(s))
}

// == Utility

const prefixTemp = "tmp-" // Prefix to add to temporary files.

// CopytoTemp creates a temporary file from the source file into the default
// directory for temporary files (see os.TempDir), whose name begins with prefix.
// If prefix is the empty string, uses the default value prefixTemp.
// Returns the temporary file name.
func CopytoTemp(source, prefix string) (tmpFile string, err error) {
	if prefix == "" {
		prefix = prefixTemp
	}

	src, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer func() {
		if err2 := src.Close(); err2 != nil && err == nil {
			err = err2
		}
	}()

	dest, err := ioutil.TempFile("", prefix)
	if err != nil {
		return "", err
	}
	defer func() {
		if err2 := dest.Close(); err2 != nil && err == nil {
			err = err2
		}
	}()

	if _, err = io.Copy(dest, src); err != nil {
		return "", err
	}

	Log.Printf("File %q copied at %q", source, dest.Name())
	return dest.Name(), nil
}
