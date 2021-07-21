// Copyright 2021 Jonas mg
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package fileutil

import (
	"io"
	"log"
)

// Logger is the global logger.
// By default, it does not write logs. Use 'SetupLogger' to logging.
var Log *log.Logger = log.New(io.Discard, "", -1)

// SetupLogger setups a logger to be used by the package.
func SetupLogger(l *log.Logger) { Log = l }
