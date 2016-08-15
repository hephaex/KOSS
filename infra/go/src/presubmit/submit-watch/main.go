// Copyright 2016 The Fuchsia Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"presubmit"
	"v.io/jiri"
	"v.io/jiri/gitutil"
	"v.io/jiri/project"
	"v.io/jiri/runutil"
	"v.io/x/lib/cmdline"
)

var (
	projectList string
	logFilePath string
	buildName   string
)

func init() {
	flag.StringVar(&projectList, "project", "", "Comma separated list of jiri projects to query")
	flag.StringVar(&logFilePath, "logfile", "/tmp/fuchsia-submit-watch-log.json", "Full path of log file to use")
	flag.StringVar(&buildName, "build", "", "The name of the build to start when new commits exist")
}

// readJiriManifest reads the jiri manifest found in JIRI_ROOT.
func readJiriManifest() (project.Projects, error) {
	jirix, err := jiri.NewX(cmdline.EnvFromOS())
	if err != nil {
		return nil, err
	}
	projects, _, err := project.LoadManifest(jirix)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func getSequence() runutil.Sequence {
	return runutil.NewSequence(nil, os.Stdin, os.Stdout, os.Stderr, false, false)
}

func getMostRecentCommit(jiriProject project.Project) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	defer os.Chdir(cwd)
	err = os.Chdir(jiriProject.Path)
	if err != nil {
		return "", err
	}

	git := gitutil.New(getSequence())
	rev, err := git.CurrentRevision()
	return rev, err
}

func readPreviousCommits() (map[string]string, error) {
	results := make(map[string]string)
	bytes, err := ioutil.ReadFile(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return results, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(bytes, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// watchNewCommits queries git for whether any new changes have been submitted since the
// last time watchNewCommits ran.  If there have been any new changes, watchNewCommits will
// kick off a Jenkins job with no arguments.
func watchNewCommits() error {
	projects, err := readJiriManifest()
	if err != nil {
		return err
	}

	// Discover the most recent commits for each project.
	projectToRevisionMap := make(map[string]string)
	for _, project := range strings.Split(projectList, ",") {
		localProject, err := projects.FindUnique(project)
		if err != nil {
			return err
		}

		rev, err := getMostRecentCommit(localProject)
		if err != nil {
			return err
		}

		projectToRevisionMap[project] = rev
	}

	// Compare the discovered map vs. the previously discovered commits.
	previousMap, err := readPreviousCommits()
	if err != nil {
		return err
	}
	if reflect.DeepEqual(previousMap, projectToRevisionMap) {
		return nil // Nothing has changed since last we checked.
	}

	// Write the new list of discovered commits.
	bytes, err := json.MarshalIndent(projectToRevisionMap, "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(logFilePath, bytes, os.FileMode(0644)); err != nil {
		return err
	}

	// Kick off the Jenkins job.
	j, err := presubmit.GetJenkins()
	if err != nil {
		return err
	}
	return j.AddBuild(buildName)
}

func main() {
	flag.Parse()

	if err := watchNewCommits(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
