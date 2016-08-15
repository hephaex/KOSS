// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package presubmit

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"v.io/jiri/gerrit"
	"v.io/jiri/runutil"
)

var (
	gerritURL   string
	GerritQuery = "status:open"
)

func init() {
	flag.StringVar(&gerritURL, "gerrit", "", "The Gerrit endpoint, e.g. https://foo-review.googlesource.com")
}

// CreateGerrit returns a handle to our gerrit instance.
func CreateGerrit() (*gerrit.Gerrit, error) {
	if len(gerritURL) == 0 {
		return nil, fmt.Errorf("No gerrit host to query; use the -gerrit flag")
	}

	// v.io/jiri/gerrit executes its commands through a v.io/jiri/runutil.Sequence object.
	//
	// runutil.Sequence contains environment variables, provides a place to override
	// std{in,out,err}, and has options for color and verbosity.  It also provides syntactic
	// sugar for executing multiple shell commands in a sequence (hence the name.)
	seq := runutil.NewSequence(nil, os.Stdin, os.Stdout, os.Stderr, false, false)

	u, err := url.Parse(gerritURL)
	if err != nil {
		return nil, err
	}

	return gerrit.New(seq, u), nil
}

type VerifiedScore int

const (
	VerifiedFail    VerifiedScore = -1
	VerifiedNeutral VerifiedScore = 0
	VerifiedPass    VerifiedScore = 1
)

func VerifiedScoreFromString(verifiedStr string) (VerifiedScore, error) {
	switch verifiedStr {
	case "-1":
		return VerifiedFail, nil
	case "0":
		return VerifiedNeutral, nil
	case "1":
		return VerifiedPass, nil
	case "+1":
		return VerifiedPass, nil
	default:
		return VerifiedNeutral, fmt.Errorf("Unrecognized 'Verified' score: %s", verifiedStr)
	}
}

// CLListToString converts a gerrit.CLList to a string for making of nice logging.
func CLListToString(cls gerrit.CLList) string {
	clRefs := []string{}
	for _, cl := range cls {
		parts := strings.Split(cl.Reference(), "/")
		if len(parts) != 5 {
			return "????/?"
		}
		clRefs = append(clRefs, fmt.Sprintf("%s/%s", parts[3], parts[4]))
	}
	return strings.Join(clRefs, ", ")
}

// PostReviewFunction interface matches gerrit.PostReview, useful for injecting stub/mock functions.
type PostReviewFunction func(ref string, msg string, labels map[string]string) error

// Post the given message to the given list of changes on Gerrit.
func PostMessageToGerrit(message string, changes gerrit.CLList, score VerifiedScore) error {
	g, err := CreateGerrit()
	if err != nil {
		return err
	}
	return InternalPostMessageToGerrit(message, changes, score,
		func(ref string, msg string, labels map[string]string) error {
			return g.PostReview(ref, msg, labels)
		})
}

// InternalPostMessageToGerrit handles the logic of setting the verified label and printing output.
// It's useful for this to be separate; during dry runs we pass a stubbed out PostReviewFunction.
func InternalPostMessageToGerrit(message string, changes gerrit.CLList, score VerifiedScore, postReview PostReviewFunction) error {
	// For all the given changes, post a review with the given message.
	for _, cl := range changes {

		// If the change uses the Verified label, set that according to the given score.
		// Some repos aren't set up to expect Verified, so we have to check first.
		var labels map[string]string
		scoreString := "N/A"
		if _, ok := cl.Labels["Verified"]; ok {
			scoreString = strconv.Itoa(int(score))
			labels = map[string]string{"Verified": scoreString}
		}

		fmt.Printf("Posting message to Gerrit (%v) %q; Verified: %s\n", CLListToString(changes), message, scoreString)
		if err := postReview(cl.Reference(), message, labels); err != nil {
			return err
		}
	}

	return nil
}
