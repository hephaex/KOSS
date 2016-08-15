// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"

	"presubmit"
	"presubmit/common"
	"v.io/jiri"
	"v.io/jiri/gerrit"
	"v.io/jiri/gitutil"
	"v.io/jiri/project"
	"v.io/jiri/runutil"
	"v.io/x/lib/cmdline"
)

const presubmitBranchName string = "underscore-presubmit"

var (
	refsToTest       string
	cleanOldBranches bool
)

func init() {
	flag.StringVar(&refsToTest, "cl", "", "Comma-separated list of change/patchset. Example: 1153/2,1150/1")
	flag.BoolVar(&cleanOldBranches, "clean", true, "Whether to remove all presubmit branches before patching")
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

// cleanAllProjects deletes any presubmit branches that already exist.  We don't run
// this after patchProject (e.g. in a defer) because unlike v23's system, our `patch`
// tool is separate and standalone.  We run this _before_ we do any patching.
//
// This approach also has the nice side effect of ensuring there are no lingering
// changes in unrelated projects, no matter what the result of previous tests.
func cleanAllProjects(allProjects project.Projects) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(cwd) // Unnecessary since we expect to quit on error, but whatev.

	git := gitutil.New(getSequence())

	for _, project := range allProjects {
		err = os.Chdir(project.Path)
		if err != nil {
			return err
		}

		if git.BranchExists(presubmitBranchName) {
			// Make sure we're not on the presubmit branch (else the delete will fail).
			err = git.CheckoutBranch("origin/master")
			if err != nil {
				return err
			}

			// Delete the presubmit branch.
			err = git.DeleteBranch(presubmitBranchName, gitutil.ForceOpt(true))
			if err != nil {
				return err
			}
		}

		// Move back to original directory in case paths are defined relatively.
		err = os.Chdir(cwd)
		if err != nil {
			return err
		}
	}
	return nil
}

// patchProject changes directory into the project directory, checks out the given
// change, then cds back to the original directory.
func patchProject(jiriProject project.Project, cl gerrit.Change) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	defer os.Chdir(cwd)
	err = os.Chdir(jiriProject.Path)
	if err != nil {
		return err
	}

	git := gitutil.New(getSequence())

	err = git.CreateAndCheckoutBranch(presubmitBranchName)
	if err != nil {
		return err
	}

	// If the pull fails, it's likely because of a merge conflict.
	err = git.Pull(jiriProject.Remote, cl.Reference())
	if err != nil {
		return err
	}

	return nil
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

	g, err := presubmit.CreateGerrit()
	quitOnError(err)

	// Construct the list of changes we're going to test.
	cls := []gerrit.Change{}
	refs, err := common.ParseRefArg(refsToTest)
	quitOnError(err)
	for _, ref := range refs {
		cl, err := g.GetChange(ref.Changelist)
		quitOnError(err)

		foundCl, foundPs, err := gerrit.ParseRefString(cl.Reference())
		quitOnError(err)

		// Abandon the test if we were given outdated patchsets.
		if foundPs != ref.Patchset {
			quitOnError(fmt.Errorf("%q is outdated; there's a newer patchset (%d/%d)\n",
				ref, foundCl, foundPs))
		}

		fmt.Printf("Found patch: %s, %s\n", cl.Project, cl.Reference())

		cls = append(cls, *cl)
	}

	// Read the project manifest.  We need this information to know which directories and remotes
	// map to the project names that come back from gerrit.
	projects, err := readJiriManifest()
	quitOnError(err)

	// Clean all projects of any previous test branches.
	if cleanOldBranches {
		err = cleanAllProjects(projects)
		quitOnError(err)
	}

	// Patch the projects that changed.
	for _, cl := range cls {
		// TODO(lanechr): remove the implicit assumption here that gerrit repo name == jiri project name.
		localProject, err := projects.FindUnique(cl.Project)
		quitOnError(err)

		err = patchProject(localProject, cl)
		quitOnError(err)
	}
}
