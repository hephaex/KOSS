// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package presubmit

import (
	"fmt"
	"os"
	"strings"

	"v.io/jiri/gerrit"
)

type CLNumber int
type Patchset int

// A Workflow handles the interaction with the Continuous Integrations system.
type Workflow interface {
	// RemoveOutdatedBuilds should halt and remove all ongoing builds that are older
	// than the given valid ones.
	RemoveOutdatedBuilds(validCLs map[CLNumber]Patchset) []error

	// AddPresubmitTestBuild should start the presubmit tests with the given CLs.
	AddPresubmitTestBuild(cls gerrit.CLList) error

	// CheckPresubmitBuildConfig returns an error if the presubmit build is not configured
	// properly, or if it fails to fetch the status of the last build.
	CheckPresubmitBuildConfig() error

	// PostResults should publish message for the given changes.  The score indicates
	// what `verified` score to assign.  Select from: Verified{Fail,Neutral,Pass}.
	PostResults(message string, changes gerrit.CLList, score VerifiedScore) error
}

// CLsSender handles the workflow and business logic of sending groups of related CLs
// to presubmit testing.  The interaction with the CI system is mocked out for testing
// and (in theory) modularity WRT adopting new CI systems.
type CLsSender struct {
	CLLists []gerrit.CLList
	CLsSent int
	Worker  Workflow
}

// SendCLstoPresubmitTest sends the set of CLLists for presubmit testing.
func (s *CLsSender) SendCLsToPresubmitTest() error {
	for _, curCLList := range s.CLLists {
		multiPartCl := combineCLList(curCLList)
		if len(multiPartCl.clMap) == 0 {
			fmt.Println("Skipping empty CL set")
			continue
		}

		// Don't send the CLs to presubmit-test if at least one of them have PresubmitTest: none.
		if multiPartCl.skipPresubmitTest {
			fmt.Printf("Skipping %s because presubmit=none\n", multiPartCl.clString)
			if err := s.Worker.PostResults(
				"Presubmit tests skipped.\n", multiPartCl.changes, VerifiedPass); err != nil {
				return err
			}
			continue
		}

		// Only test code submitted by trusted contributors.
		if !multiPartCl.hasTrustedOwner {
			fmt.Printf("Skipping %s because the owner is an external contributor\n", multiPartCl.clString)
			if err := s.Worker.PostResults(
				"Tell Freenode#fuchsia to kick the presubmit tests.\n", multiPartCl.changes, VerifiedFail); err != nil {
				return err
			}
			continue
		}

		// Cancel any previous tests from old patch sets that may still be running.
		for _, err := range s.Worker.RemoveOutdatedBuilds(multiPartCl.clMap) {
			if err != nil {
				fmt.Fprintln(os.Stderr, err) // Not fatal; just log errors.
			}
		}

		// Finally send the CLs to presubmit-test.
		fmt.Printf("Sending %s to presubmit test\n", multiPartCl.clString)
		if err := s.Worker.AddPresubmitTestBuild(curCLList); err != nil {
			fmt.Fprintf(os.Stderr, "addPresubmitTestBuild failed: %v\n", err)
		} else {
			s.CLsSent += len(curCLList)
		}

		// Notify the author that their change(s) have been sent for presubmit testing.
		if err := s.Worker.PostResults(
			"Change was sent for presubmit testing.  Please stand by.\n", multiPartCl.changes, VerifiedNeutral); err != nil {
			return err
		}
	}
	return nil
}

// multiPartCLInfo collects all the data about a list of CLs that we run at the same time.
// Because a single logical change may be broken up into multiple individual CLs, we have to
// run tests on many CLs at once.  Colloquially this is referred to as a "multi part" CL.
type multiPartCLInfo struct {
	clMap             map[CLNumber]Patchset
	clString          string
	skipPresubmitTest bool
	hasTrustedOwner   bool
	changes           gerrit.CLList
}

// combineCLList combines the given individual CLs into a single multiPartCLInfo.
func combineCLList(curCLList gerrit.CLList) multiPartCLInfo {
	result := multiPartCLInfo{}
	result.hasTrustedOwner = true
	result.clMap = map[CLNumber]Patchset{}
	clStrings := []string{}

	for _, curCL := range curCLList {
		// If we have a malformed ref string, we can't recover.  Must abort.
		cl, ps, err := gerrit.ParseRefString(curCL.Reference())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return multiPartCLInfo{}
		}

		// Check if the author has indicated this change should avoid presubmit tests.
		if curCL.PresubmitTest == gerrit.PresubmitTestTypeNone {
			result.skipPresubmitTest = true
		}

		// If any of the CLs aren't trusted, mark the whole list as untrusted.
		if !isTrustedContributor(curCL.OwnerEmail()) {
			result.hasTrustedOwner = false
		}

		clStrings = append(clStrings, formatCLString(cl, ps))
		result.clMap[CLNumber(cl)] = Patchset(ps)
		result.changes = append(result.changes, curCL)
	}

	result.clString = strings.Join(clStrings, ", ")
	return result
}

// isTrustedContributor returns whether this owner is a "trusted" contributor.  Being trusted
// controls whether we automatically run your code through tests.  Currently this function
// just checks if you're submitting from a google.com address.  In the future, it could use an
// ACL or something.
func isTrustedContributor(emailAddress string) bool {
	return strings.HasSuffix(emailAddress, "@google.com")
}

// formatCLString formats the given cl and patch numbers into a user-readable description.  V23
// does this as a URL like http://go/vcl/xxxx/yy.  We could do something similar, but don't!
func formatCLString(clNumber int, patchsetNumber int) string {
	return fmt.Sprintf("%d/%d", clNumber, patchsetNumber)
}
