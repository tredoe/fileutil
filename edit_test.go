// Copyright 2012 Jonas mg
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fileutil

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func TestCreate(t *testing.T) {
	if err := CreateString(fileTemp, `
  Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor 
incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis 
nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. 
Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu 
fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in 
culpa qui officia deserunt mollit anim id est laborum.
`); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("wc", "-l", fileTemp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != '7' {
		t.Fatalf("got %q lines, want 7", out[0])
	}
}

func TestEdit(t *testing.T) {
	line := "I've heard that the night is all magic.\n"

	ed, err := NewEdit(fileTemp, &ConfEditer{Comment: []byte{'#'}, Mode: ModBackup})
	if err != nil {
		t.Fatal(err)
	}
	/*defer func() {
		if err = os.Remove(fileTemp); err != nil {
			t.Error(err)
		}
	}()*/
	defer func() {
		if err = ed.Close(); err != nil {
			t.Error(err)
		}
	}()

	// The backup should be created.
	if _, err = os.Stat(fileBackup); err != nil {
		t.Error(err)
	}
	defer func() {
		if err = os.Remove(fileBackup); err != nil {
			t.Error(err)
		}
	}()

	// Append
	if err = ed.AppendString("\n" + line); err != nil {
		t.Error(err)
	} else {
		cmd := exec.Command("tail", "-n1", fileTemp)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != line {
			t.Errorf("Append => got %q, want %q", out, line)
		}
	}

	/*// Insert
	if err = ed.InsertString(line); err != nil {
		t.Error(err)
	} else {
		cmd := exec.Command("head", "-n1", fileTemp)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != line {
			t.Errorf("Insert => got %q, want %q", out, line)
		}
	}*/

	// Replace
	repl := []Replacer{
		{"dolor", "DOL_"},
		{"labor", "LABOR_"},
	}
	resul := "3\n"

	if err = ed.Replace(repl); err != nil {
		t.Error(err)
	} else {
		cmd := exec.Command("grep", "-c", repl[1].Replace, fileTemp)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != resul {
			t.Errorf("Replace (%s) => got %v, want %v", repl[1].Replace, out, resul)
		}
	}

	repl = []Replacer{
		{"DOL_", "dOlOr"},
		{"LABOR_", "lAbOr"},
	}
	resul = "1\n"

	if err = ed.ReplaceN(repl, 1); err != nil {
		t.Error(err)
	} else {
		for i := 0; i <= 1; i++ {
			cmd := exec.Command("grep", "-c", repl[i].Replace, fileTemp)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != resul {
				t.Errorf("Replace (%s) => got %v, want %v", repl[i].Replace, out, resul)
			}
		}
	}

	// ReplaceAtLine
	replAt := []ReplacerAtLine{
		{"LABOR", "o", "OO"},
	}
	resul = "2\n"

	if err = ed.ReplaceAtLine(replAt); err != nil {
		t.Error(err)
	} else {
		cmd := exec.Command("grep", "-c", "OO", fileTemp)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != resul {
			t.Errorf("ReplaceAtLine => got %v, want %v", out, resul)
		}
	}

	replAt = []ReplacerAtLine{
		{"heard", "a", "AA"},
	}
	resul = "1\n"

	if err = ed.ReplaceAtLineN(replAt, 2); err != nil {
		t.Error(err)
	} else {
		var out bytes.Buffer
		cmd1 := exec.Command("tail", "-n1", fileTemp)
		cmd2 := exec.Command("grep", "-c", "A")

		if cmd2.Stdin, err = cmd1.StdoutPipe(); err != nil {
			t.Fatal(err)
		}
		cmd2.Stdout = &out

		if err = cmd2.Start(); err != nil {
			t.Fatal(err)
		}
		if err = cmd1.Run(); err != nil {
			t.Fatal(err)
		}
		if err = cmd2.Wait(); err != nil {
			t.Fatal(err)
		}

		if out.String() != resul {
			t.Errorf("ReplaceAtLineN => got %v, want %v", out, resul)
		}
	}

	// Comment
	resul = "2\n"

	if err = ed.Comment([]string{"night", "quis"}); err != nil {
		t.Error(err)
	} else {
		cmd := exec.Command("grep", "-c", string(ed.conf.Comment), fileTemp)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != resul {
			t.Errorf("Comment => got %v, want %v", out, resul)
		}
	}

	// CommentOut
	resul = "0\n"

	if err = ed.CommentOut([]string{"night", "quis"}); err != nil {
		t.Error(err)
	} else {
		cmd := exec.Command("grep", "-c", string(ed.conf.Comment), fileTemp)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// At 'grep', normally the exit status is 0 if a line is selected, 1 if no lines were
			// selected, and 2 if an error occurred.
			typErr := err.(*exec.ExitError)
			if typErr.ExitCode() == 2 {
				t.Fatal(err)
			}
		}
		if string(out) != resul {
			t.Errorf("CommentOut => got %v, want %v", out, resul)
		}
	}
}
