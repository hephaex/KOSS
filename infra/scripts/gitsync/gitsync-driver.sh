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

# Fetch the list of Fuchsia repos to sync and run a gitsync.sh for each.
# Designed to be run unattended via cron job.

SRC_HOST="https://fuchsia.googlesource.com"
DST_HOST="https://github.com/fuchsia-mirror"
SCRIPT_DIR="$( cd $( dirname ${BASH_SOURCE[0]} ) && pwd)"
LOG_DIR="$SCRIPT_DIR/log/$(date +%Y)/$(date +%m)/$(date +%d)"
LOG_FILE="$LOG_DIR/$(date +%H%M%S).log"

cd "${SCRIPT_DIR}"
mkdir -p "${LOG_DIR}"

# Everything below the next line will be logged to $LOG_FILE.
exec 1>"${LOG_FILE}" 2>&1
set -x
echo "[gitsync] Start log"

# Sync fnl-start to pull down the latest list of repos.
git pull

# We keep a list of repos checked in.  We could maybe use GitHub APIs instead.
REPO_LIST=$(cat "${SCRIPT_DIR}"/list-of-repos.txt)
for repo in ${REPO_LIST}; do
  echo "[gitsync] Syncing ${repo}..."
  "${SCRIPT_DIR}"/gitsync.sh "${SRC_HOST}" "${DST_HOST}" "${repo}"
  date +%H:%M:%S
done

echo "[gitsync] Exited normally"
