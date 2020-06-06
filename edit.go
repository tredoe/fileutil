// Copyright 2012 Jonas mg
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fileutil

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"regexp"
)

// A ModeEdit value is a set of flags (or 0) to control behavior at edit a file.
type ModeEdit uint

// Modes used at edit a file.
const (
	_         ModeEdit = iota
	ModBackup          // Do backup before of edit.
)

// ConfEditer represents the editer configuration.
type ConfEditer struct {
	Comment []byte
	Mode    ModeEdit
}

// Editer represents the file to edit.
type Editer struct {
	file *os.File
	buf  *bufio.ReadWriter
	conf *ConfEditer
}

// NewEdit prepares a file to edit.
// You must use 'Close()' to close the file.
func NewEdit(filename string, conf *ConfEditer) (*Editer, error) {
	if conf != nil && conf.Mode&ModBackup != 0 {
		if err := Backup(filename); err != nil {
			return nil, err
		}
	}

	file, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	return &Editer{
		file: file,
		buf:  bufio.NewReadWriter(bufio.NewReader(file), bufio.NewWriter(file)),
		conf: conf,
	}, nil
}

// Close closes the file.
func (ed *Editer) Close() error {
	return ed.file.Close()
}

// Append writes len(b) bytes at the end of the File. It returns an error, if any.
func (ed *Editer) Append(b []byte) error {
	_, err := ed.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	_, err = ed.file.Write(b)
	return err
}

// AppendString is like Append, but writes the contents of string s rather than an array of bytes.
func (ed *Editer) AppendString(s string) error {
	return ed.Append([]byte(s))
}

// Delete removes the text given at position 'begin:end'.
func (ed *Editer) Delete(begin, end int64) error {
	stat, err := ed.file.Stat()
	if err != nil {
		return err
	}

	if _, err = ed.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	fileData := make([]byte, begin)

	if _, err = ed.file.Read(fileData); err != nil {
		return err
	}
	buf.Write(fileData)

	//fileData = fileData[:]
	fileData = make([]byte, stat.Size()-(end-begin))

	if _, err = ed.file.Seek(end, io.SeekStart); err != nil {
		return err
	}
	if _, err = ed.file.Read(fileData); err != nil {
		return err
	}
	buf.Write(fileData)
	fileData = fileData[:]

	return ed.rewrite(buf.Bytes())
}

// Comment inserts the comment character in lines that mach any regular expression in reLine.
func (ed *Editer) Comment(reLine []string) error {
	if ed.conf == nil || len(ed.conf.Comment) == 0 {
		return errComment
	}

	allReSearch := make([]*regexp.Regexp, len(reLine))

	for i, v := range reLine {
		re, err := regexp.Compile(v)
		if err != nil {
			return err
		}

		allReSearch[i] = re
	}

	_, err := ed.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	char := append(ed.conf.Comment, ' ')
	isNew := false
	buf := new(bytes.Buffer)

	// Check every line.
	for {
		line, err := ed.buf.ReadBytes('\n')
		if err == io.EOF {
			break
		}

		for _, v := range allReSearch {
			if v.Match(line) {
				line = append(char, line...)

				if !isNew {
					isNew = true
				}
				break
			}
		}

		if _, err = buf.Write(line); err != nil {
			return err
		}
	}

	if isNew {
		return ed.rewrite(buf.Bytes())
	}
	return nil
}

// Replacer repreents the text to be replaced.
type Replacer struct {
	Search, Replace string
}

// ReplacerAtLine repreents the text to be replaced into a line.
type ReplacerAtLine struct {
	Line, Search, Replace string
}

// CommentOut removes the comment character of lines that mach any regular expression in reLine.
func (ed *Editer) CommentOut(reLine []string) error {
	if ed.conf == nil || len(ed.conf.Comment) == 0 {
		return errComment
	}
	allSearch := make([]ReplacerAtLine, len(reLine))

	for i, v := range reLine {
		allSearch[i] = ReplacerAtLine{
			v, "[[:space:]]*" + string(ed.conf.Comment) + "[[:space:]]*", "",
		}
	}

	return ed.ReplaceAtLineN(allSearch, 1)
}

/*// Insert writes len(b) bytes at the start of the File. It returns an error, if any.
func (ed *Editer) Insert(b []byte) error {
	return ed.rewrite(b)
}

// InsertString is like Insert, but writes the contents of string s rather than an array of bytes.
func (ed *Editer) InsertString(s string) error {
	return ed.rewrite([]byte(s))
}*/

// Replace replaces all regular expressions mathed in r.
func (ed *Editer) Replace(r []Replacer) error {
	return ed.genReplace(r, -1)
}

// ReplaceN replaces regular expressions mathed in r. The count determines the number to match:
//   n > 0: at most n matches
//   n == 0: the result is none
//   n < 0: all matches
func (ed *Editer) ReplaceN(r []Replacer, n int) error {
	return ed.genReplace(r, n)
}

// ReplaceAtLine replaces all regular expressions mathed in r, if the line is matched at the first.
func (ed *Editer) ReplaceAtLine(r []ReplacerAtLine) error {
	return ed.genReplaceAtLine(r, -1)
}

// ReplaceAtLineN replaces regular expressions mathed in r, if the line is matched at the first.
// The count determines the number to match:
//   n > 0: at most n matches
//   n == 0: the result is none
//   n < 0: all matches
func (ed *Editer) ReplaceAtLineN(r []ReplacerAtLine, n int) error {
	return ed.genReplaceAtLine(r, n)
}

// Generic Replace: replaces a number of regular expressions matched in r.
func (ed *Editer) genReplace(r []Replacer, n int) error {
	if n == 0 {
		return nil
	}
	if _, err := ed.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	content, err := ioutil.ReadAll(ed.buf)
	if err != nil {
		return err
	}

	isNew := false

	for _, v := range r {
		reSearch, err := regexp.Compile(v.Search)
		if err != nil {
			return err
		}

		i := n
		repl := []byte(v.Replace)

		content = reSearch.ReplaceAllFunc(content, func(s []byte) []byte {
			if !isNew {
				isNew = true
			}

			if i != 0 {
				i--
				return repl
			}
			return s
		})
	}

	if isNew {
		return ed.rewrite(content)
	}
	return nil
}

// Generic ReplaceAtLine: replaces a number of regular expressions matched in r,
// if the line is matched at the first.
func (ed *Editer) genReplaceAtLine(r []ReplacerAtLine, n int) error {
	if n == 0 {
		return nil
	}
	_, err := ed.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	// == Cache the regular expressions
	allReLine := make([]*regexp.Regexp, len(r))
	allReSearch := make([]*regexp.Regexp, len(r))
	allRepl := make([][]byte, len(r))

	for i, v := range r {
		reLine, err := regexp.Compile(v.Line)
		if err != nil {
			return err
		}
		allReLine[i] = reLine

		reSearch, err := regexp.Compile(v.Search)
		if err != nil {
			return err
		}
		allReSearch[i] = reSearch

		allRepl[i] = []byte(v.Replace)
	}

	buf := new(bytes.Buffer)
	isNew := false

	// Replace every line, if it maches
	for {
		line, err := ed.buf.ReadBytes('\n')
		if err == io.EOF {
			break
		}

		for i := range r {
			if allReLine[i].Match(line) {
				j := n

				line = allReSearch[i].ReplaceAllFunc(line, func(s []byte) []byte {
					if !isNew {
						isNew = true
					}

					if j != 0 {
						j--
						return allRepl[i]
					}
					return s
				})
			}
		}
		if _, err = buf.Write(line); err != nil {
			return err
		}
	}

	if isNew {
		return ed.rewrite(buf.Bytes())
	}
	return nil
}

func (ed *Editer) rewrite(b []byte) error {
	_, err := ed.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	n, err := ed.file.Write(b)
	if err != nil {
		return err
	}
	if err = ed.file.Truncate(int64(n)); err != nil {
		return err
	}
	return nil // ed.file.Sync()
}

// * * *

// Append writes len(b) bytes at the end of the named file.
// It returns an error, if any.
func Append(filename string, mode ModeEdit, b []byte) error {
	ed, err := NewEdit(filename, &ConfEditer{Mode: mode})
	if err != nil {
		return err
	}

	err = ed.Append(b)
	err2 := ed.Close()
	if err != nil {
		return err
	}
	return err2
}

// AppendString is like Append, but writes the contents of string s rather than an array of bytes.
func AppendString(filename string, mode ModeEdit, s string) error {
	return Append(filename, mode, []byte(s))
}

// Comment inserts the comment character in lines that mach the regular expression in reLine,
// in the named file.
func Comment(filename string, conf *ConfEditer, reLine string) error {
	return CommentM(filename, conf, []string{reLine})
}

// CommentM inserts the comment character in lines that mach any regular expression in reLine,
// in the named file.
func CommentM(filename string, conf *ConfEditer, reLine []string) error {
	ed, err := NewEdit(filename, conf)
	if err != nil {
		return err
	}

	err = ed.Comment(reLine)
	err2 := ed.Close()
	if err != nil {
		return err
	}
	return err2
}

// CommentOut removes the comment character of lines that mach the regular expression in reLine,
// in the named file.
func CommentOut(filename string, conf *ConfEditer, reLine string) error {
	return CommentOutM(filename, conf, []string{reLine})
}

// CommentOutM removes the comment character of lines that mach any regular expression in reLine,
// in the named file.
func CommentOutM(filename string, conf *ConfEditer, reLine []string) error {
	ed, err := NewEdit(filename, conf)
	if err != nil {
		return err
	}

	err = ed.CommentOut(reLine)
	err2 := ed.Close()
	if err != nil {
		return err
	}
	return err2
}

/*// Insert writes len(b) bytes at the start of the named file. It returns an error, if any.
func Insert(filename string, conf *ConfEditer, b []byte) error {
	ed, err := NewEdit(filename, conf)
	if err != nil {
		return err
	}

	err = ed.Insert(b)
	err2 := ed.Close()
	if err != nil {
		return err
	}
	return err2
}

// InsertString is like Insert, but writes the contents of string s rather than an array of bytes.
func InsertString(filename string, conf *ConfEditer, s string) error {
	return Insert(filename, conf, []byte(s))
}*/

// Replace replaces all regular expressions mathed in r for the named file.
func Replace(filename string, conf *ConfEditer, r []Replacer) error {
	ed, err := NewEdit(filename, conf)
	if err != nil {
		return err
	}

	err = ed.genReplace(r, -1)
	err2 := ed.Close()
	if err != nil {
		return err
	}
	return err2
}

// ReplaceN replaces a number of regular expressions mathed in r for the named file.
func ReplaceN(filename string, conf *ConfEditer, r []Replacer, n int) error {
	ed, err := NewEdit(filename, conf)
	if err != nil {
		return err
	}

	err = ed.genReplace(r, n)
	err2 := ed.Close()
	if err != nil {
		return err
	}
	return err2
}

// ReplaceAtLine replaces all regular expressions mathed in r for the named file,
// if the line is matched at the first.
func ReplaceAtLine(filename string, conf *ConfEditer, r []ReplacerAtLine) error {
	ed, err := NewEdit(filename, conf)
	if err != nil {
		return err
	}

	err = ed.genReplaceAtLine(r, -1)
	err2 := ed.Close()
	if err != nil {
		return err
	}
	return err2
}

// ReplaceAtLineN replaces a number of regular expressions mathed in r for the named file,
// if the line is matched at the first.
func ReplaceAtLineN(filename string, conf *ConfEditer, r []ReplacerAtLine, n int) error {
	ed, err := NewEdit(filename, conf)
	if err != nil {
		return err
	}

	err = ed.genReplaceAtLine(r, n)
	err2 := ed.Close()
	if err != nil {
		return err
	}
	return err2
}
