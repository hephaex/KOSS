// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"presubmit"
	"presubmit/common"
	"v.io/jiri/gerrit"
)

var (
	refsToUpdate string
	verifiedArg  string
	messageFile  string
)

func init() {
	flag.StringVar(&refsToUpdate, "cl", "", "Comma-separated list of change/patchset. Example: 1153/2,1150/1")
	flag.StringVar(&verifiedArg, "verified", "0", "The value to set on the Verified label. Choose from: -1,0,+1")
	flag.StringVar(&messageFile, "file", "", "A file containing a message to write")
}

// quitOnError is a convenience function for printing an error and exiting the program.
func quitOnError(e error) {
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
}

func main() {
	flag.Parse()

	// Read the message.
	bytes, err := ioutil.ReadFile(messageFile)
	quitOnError(err)
	message := string(bytes)

	// Verify the value given to -verified.
	score, err := presubmit.VerifiedScoreFromString(verifiedArg)
	quitOnError(err)

	// Make a Gerrit!
	g, err := presubmit.CreateGerrit()
	quitOnError(err)

	// Convert the list of cls to gerrit.Change objects.
	refs, err := common.ParseRefArg(refsToUpdate)
	quitOnError(err)
	cls := gerrit.CLList{}
	for _, ref := range refs {
		cl, err := g.GetChange(ref.Changelist)
		quitOnError(err)
		cls = append(cls, *cl)
	}

	// Post them message to that thar Gerrit.
	err = presubmit.PostMessageToGerrit(message, cls, score)
	quitOnError(err)
}
