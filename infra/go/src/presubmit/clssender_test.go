// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package presubmit

import (
	"reflect"
	"testing"
	"v.io/jiri/gerrit"
)

type postedResult struct {
	message  string
	refs     []string
	verified VerifiedScore
}

type stubWorkflow struct {
	postMessageRecord []postedResult
	clsAddedForTest   []gerrit.CLList
}

func (mw *stubWorkflow) RemoveOutdatedBuilds(outdatedCLs map[CLNumber]Patchset) []error {
	return nil
}

func (mw *stubWorkflow) AddPresubmitTestBuild(cls gerrit.CLList) error {
	mw.clsAddedForTest = append(mw.clsAddedForTest, cls)
	return nil
}

func (mw *stubWorkflow) PostResults(message string, changes gerrit.CLList, score VerifiedScore) error {
	clRefs := []string{}
	for _, cl := range changes {
		clRefs = append(clRefs, cl.Reference())
	}
	mw.postMessageRecord = append(mw.postMessageRecord, postedResult{message, clRefs, score})
	return nil
}

func (mw *stubWorkflow) CheckPresubmitBuildConfig() error {
	return nil
}

func TestSendCLsToPresubmitTest(t *testing.T) {
	clLists := []gerrit.CLList{
		// Expect this to be run.
		gerrit.CLList{
			gerrit.GenCL(1000, 1, "release.js.core"),
		},

		// Expect this to be ignored because PresubmitTestTypeNone is set.
		gerrit.CLList{
			gerrit.GenCLWithMoreData(2000, 1, "release.js.core", gerrit.PresubmitTestTypeNone, "vj@google.com"),
		},

		// Expect this to be skipped because external contributor.
		gerrit.CLList{
			gerrit.GenCLWithMoreData(2010, 1, "release.js.core", gerrit.PresubmitTestTypeAll, "foo@bar.com"),
		},

		// Expect these to be run.
		gerrit.CLList{
			gerrit.GenMultiPartCL(1001, 1, "release.js.core", "t", 1, 2),
			gerrit.GenMultiPartCL(1002, 1, "release.go.core", "t", 2, 2),
		},

		// Expect all of these to be skipped because one of them has an external contributor.
		gerrit.CLList{
			gerrit.GenMultiPartCL(1003, 1, "release.js.core", "t", 1, 3),
			gerrit.GenMultiPartCL(1004, 1, "release.go.core", "t", 2, 3),
			gerrit.GenMultiPartCLWithMoreData(1005, 1, "release.go.core", "t", 3, 3, "foo@bar.com"),
		},
	}

	stubWorker := stubWorkflow{}
	sender := CLsSender{
		CLLists: clLists,
		Worker:  &stubWorker,
	}
	if err := sender.SendCLsToPresubmitTest(); err != nil {
		t.Fatalf("sendCLsToPresubmitTest returned error: %v", err)
	}

	if got, want := sender.CLsSent, 3; got != want {
		t.Fatalf("numSentCLs: got %d, want %d", got, want)
	}

	expectedCLsSent := []gerrit.CLList{clLists[0], clLists[3]}
	if !reflect.DeepEqual(expectedCLsSent, stubWorker.clsAddedForTest) {
		t.Fatalf("cls sent for results: got %v, want %v", stubWorker.clsAddedForTest, expectedCLsSent)
	}

	expectedPostedResults := []postedResult{
		postedResult{"Change was sent for presubmit testing.  Please stand by.\n", []string{"refs/changes/xx/1000/1"}, VerifiedNeutral},
		postedResult{"Presubmit tests skipped.\n", []string{"refs/changes/xx/2000/1"}, VerifiedPass},
		postedResult{"Tell Freenode#fuchsia to kick the presubmit tests.\n", []string{"refs/changes/xx/2010/1"}, VerifiedFail},
		postedResult{"Change was sent for presubmit testing.  Please stand by.\n", []string{"refs/changes/xx/1001/1", "refs/changes/xx/1002/1"}, VerifiedNeutral},
		postedResult{"Tell Freenode#fuchsia to kick the presubmit tests.\n", []string{
			"refs/changes/xx/1003/1",
			"refs/changes/xx/1004/1",
			"refs/changes/xx/1005/1",
		}, VerifiedFail},
	}
	if !reflect.DeepEqual(expectedPostedResults, stubWorker.postMessageRecord) {
		t.Fatalf("posted messages: got %v, want %v", stubWorker.postMessageRecord, expectedPostedResults)
	}
}
