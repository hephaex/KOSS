// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package presubmit

// This file contains all the functions that contain Jenkins-specific logic.  Other
// files should be able to reference these functions without knowing what CI system
// is backing them.  No other files should even mention Jenkins.

import (
	"flag"
	"net/url"
	"strings"

	"v.io/jiri/gerrit"
	"v.io/jiri/jenkins"
)

var (
	jenkinsURL      string
	jenkinsInstance *jenkins.Jenkins
)

func init() {
	flag.StringVar(&jenkinsURL, "jenkins", "http://localhost:8001/jenkins", "The Jenkins endpoint")
}

// GetJenkins returns a handle to the Jenkins instance in a non-thread-safe singleton fashion.
func GetJenkins() (*jenkins.Jenkins, error) {
	if jenkinsInstance != nil {
		return jenkinsInstance, nil
	}
	var err error
	jenkinsInstance, err = jenkins.New(jenkinsURL)
	return jenkinsInstance, err
}

// CheckPresubmitBuildConfig returns an error if the presubmit build is not configured properly.
// It also returns an error if we fail to fetch the status of the build.
func CheckPresubmitBuildConfig(testName string) error {
	j, err := GetJenkins()
	if err != nil {
		return err
	}

	_, err = j.LastCompletedBuildStatus(testName, nil)
	if err != nil {
		return err
	}

	return nil
}

// RemoveOutdatedBuilds halts and removes presubmit builds that are no longer relevant.  This
// could happen because a contributor uploads a new patch set before the old one is finished testing.
func RemoveOutdatedBuilds(validCLs map[CLNumber]Patchset) (errs []error) {
	// TODO(lanechr): everything.
	return nil
}

// AddPresubmitTestBuild kicks off the presubmit test build on Jenkins.
func AddPresubmitTestBuild(testName string, cls gerrit.CLList) error {
	j, err := GetJenkins()
	if err != nil {
		return err
	}

	refs := []string{}
	for _, cl := range cls {
		refs = append(refs, cl.Reference())
	}

	// The presubmit test must be parameterized to expect comma-separated refs.
	err = j.AddBuildWithParameter(testName, url.Values{"REFS": {strings.Join(refs, ",")}})
	if err != nil {
		return err
	}

	return nil
}
