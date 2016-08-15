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

# This script is used by the VanadiumSCM Jenkins plugin to initialize jiri.

set -x
set -e

# Make sure $JIRI_ROOT isn't blank because we're going to use it later to construct paths.
if [[ -z "${JIRI_ROOT}" ]]; then
  echo "JIRI_ROOT must be set!"
  exit 99
fi

# Download jiri if it doesn't already exist.
if [[ ! -e "${JIRI_ROOT}/.jiri_root/bin/jiri" ]]; then
  curl -s https://raw.githubusercontent.com/vanadium/go.jiri/master/scripts/bootstrap_jiri | bash -s "${JIRI_ROOT}"
fi
