// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"fmt"
	"strconv"
	"strings"
)

type ReferenceArg struct {
	Changelist int
	Patchset   int
}

func (ra ReferenceArg) String() string {
	return fmt.Sprintf("%d/%d", ra.Changelist, ra.Patchset)
}

// ParseRefArg parses the argument given to the -cl flag.
func ParseRefArg(commaSeparatedRefs string) ([]ReferenceArg, error) {
	result := []ReferenceArg{}
	for _, ref := range strings.Split(commaSeparatedRefs, ",") {
		parts := strings.Split(ref, "/")
		if len(parts) != 2 {
			// Allow for ref strings in the form of a gerrit reference.
			if strings.HasPrefix(ref, "refs/changes/") && len(parts) == 5 {
				parts = parts[3:]
			} else {
				return nil, fmt.Errorf(
					"malformed cl string: %q; examples of supported forms are: 'refs/changes/53/1153/2', or '1153/2'\n", ref)
			}
		}
		cl, e := strconv.Atoi(parts[0])
		if e != nil {
			return nil, e
		}
		ps, e := strconv.Atoi(parts[1])
		if e != nil {
			return nil, e
		}
		result = append(result, ReferenceArg{cl, ps})
	}
	return result, nil
}
