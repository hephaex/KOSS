// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"presubmit"
	"v.io/jiri/gerrit"
)

var (
	repoList    string
	logFilePath string
	forceSend   bool
	dryRun      bool
	testName    string
)

func init() {
	flag.StringVar(&repoList, "repo", "", "Comma separated list of repos to query")
	flag.StringVar(&logFilePath, "logfile", "/tmp/fuchsia-presubmit-log.json", "Full path of log file to use")
	flag.BoolVar(&forceSend, "f", false, "Send all changes, even if they've already been sent")
	flag.BoolVar(&dryRun, "n", false, "Query gerrit, log normally, but don't actually send to CI")
	flag.StringVar(&testName, "test", "presubmit-test", "The name of the presubmit test job")
}

// sendNewChangesForTesting queries gerrit for new changes, where changes may be grouped into related
// sets that must be tested together (i.e. MultiPart changes.), then sends them for testing.
func sendNewChangesForTesting() error {
	numberOfSentCLs := 0
	defer func() {
		fmt.Printf("Sent %d CLs for testing\n", numberOfSentCLs)
	}()

	// Grab handle to Gerrit.
	gerritHandle, err := presubmit.CreateGerrit()
	if err != nil {
		return err
	}

	// Pick a CI worker.
	var worker presubmit.Workflow
	if dryRun {
		fmt.Println("DRY RUN (will not send anything to CI, or write to Gerrit)")
		worker = &DryRunCIWorker{}
	} else {
		worker = &JenkinsGerritCIWorker{}
	}

	// Don't send any changes if the presubmit test job is not configured properly.  This is
	// nice because `query` will keep track of the CLs it sends to presubmit and will not send
	// them more than once.  So this makes sure we're sending to a job that at least looks ready.
	if err := worker.CheckPresubmitBuildConfig(); err != nil {
		return fmt.Errorf("Refusing to test new CLs because of existing failures\n%v", err)
	}

	// Read previously found CLs.
	var prevCLsMap gerrit.CLRefMap
	if !forceSend {
		fmt.Println("Using CL log:", logFilePath)
		prevCLsMap, err = gerrit.ReadLog(logFilePath)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("Force sending all pending changes")
	}

	// Fetch pending CLs from Gerrit.
	pendingCLs, err := gerritHandle.Query(presubmit.GerritQuery)
	if err != nil {
		return err
	}

	// Filter the list of CLs by repo.
	var filteredCLs gerrit.CLList
	if len(repoList) != 0 {
		repoFilterList := map[string]bool{}
		for _, repo := range strings.Split(repoList, ",") {
			repoFilterList[repo] = true
		}

		for _, cl := range pendingCLs {
			if repoFilterList[cl.Project] {
				filteredCLs = append(filteredCLs, cl)
			}
		}
	} else {
		filteredCLs = pendingCLs
	}

	// Write the current list of pending CLs to a file.
	err = gerrit.WriteLog(logFilePath, filteredCLs)
	if err != nil {
		return err
	}

	// Compare the previous CLs to the current list to determine which new CLs we must
	// send for testing.
	newCLs, errList := gerrit.NewOpenCLs(prevCLsMap, filteredCLs)
	errMsg := ""
	for _, e := range errList {
		// NewOpenCLs may return errors detected when parsing MultiPart CL metadata.
		errMsg += fmt.Sprintf("NewOpenCLs error: %v\n", e)
	}
	if len(errMsg) > 0 {
		return fmt.Errorf(errMsg)
	}

	// Send the CLs for testing.
	sender := presubmit.CLsSender{
		CLLists: newCLs,
		Worker:  worker,
	}
	if err := sender.SendCLsToPresubmitTest(); err != nil {
		return err
	}

	numberOfSentCLs = sender.CLsSent
	return nil
}

// JenkinsGerritCIWorker implements a workflow for clsSender with jenkins as CI and gerrit for code review.
type JenkinsGerritCIWorker struct{}

func (jg *JenkinsGerritCIWorker) RemoveOutdatedBuilds(outdatedCLs map[presubmit.CLNumber]presubmit.Patchset) []error {
	return presubmit.RemoveOutdatedBuilds(outdatedCLs)
}

func (jg *JenkinsGerritCIWorker) AddPresubmitTestBuild(cls gerrit.CLList) error {
	return presubmit.AddPresubmitTestBuild(testName, cls)
}

func (jg *JenkinsGerritCIWorker) CheckPresubmitBuildConfig() error {
	return presubmit.CheckPresubmitBuildConfig(testName)
}

func (jg *JenkinsGerritCIWorker) PostResults(message string, changes gerrit.CLList, score presubmit.VerifiedScore) error {
	return presubmit.PostMessageToGerrit(message, changes, score)
}

// DryRunCIWorker implements a workflow for clsSender that does not try to do any CI.  Useful if
// you're not running a Jenkins instance on your localhost.
type DryRunCIWorker struct{}

func (w *DryRunCIWorker) RemoveOutdatedBuilds(outdatedCLs map[presubmit.CLNumber]presubmit.Patchset) []error {
	return nil
}

func (w *DryRunCIWorker) AddPresubmitTestBuild(cls gerrit.CLList) error {
	return nil
}

func (w *DryRunCIWorker) CheckPresubmitBuildConfig() error {
	return nil
}

func (w *DryRunCIWorker) PostResults(message string, changes gerrit.CLList, score presubmit.VerifiedScore) error {
	return presubmit.InternalPostMessageToGerrit(message, changes, score,
		func(ref string, msg string, labels map[string]string) error {
			return nil
		})
}

func main() {
	flag.Parse()

	if err := sendNewChangesForTesting(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
