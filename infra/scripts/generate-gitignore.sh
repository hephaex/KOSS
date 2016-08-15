#!/bin/bash

# Copyright 2016 The Fuchsia Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script generates a .gitignore that you can use for new repos.

# Usage: ./generate-gitignore.bash will write a new .gitignore in the cwd,
# but only if there isn't already a .gitignore.

set -e

if [[ -f ".gitignore" ]]; then
  echo "Refusing to overwrite existing .gitignore.  Delete it first, please."
  exit 1
fi

set -x

touch .gitignore
PREFIXES=(Vim Emacs Linux OSX)
for PREFIX in ${PREFIXES[@]}; do
  curl -f https://raw.githubusercontent.com/github/gitignore/master/Global/$PREFIX.gitignore >> .gitignore
done
