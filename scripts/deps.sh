#!/bin/bash -e

#http://www.apache.org/licenses/LICENSE-2.0.txt
#
#
#Copyright 2015 Intel Corporation
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

# This script runs the correct godep sequences for snap and built-in plugins
# This will rebase back to the committed version. It should be run from snap/.
ctrl_c()
{
  exit $?
}
trap ctrl_c SIGINT

# First load snap deps
echo "Installing dependencies if not exists"
go get github.com/tools/godep
go get github.com/intelsdi-x/snap/control

echo "Checking snap root for deps"
godep restore
# REST API
echo "Checking snapctl for deps"
cd cmd/snapctl
godep restore
# CLI
echo "Checking snap mgmt/rest for deps"
cd ../../mgmt/rest
godep restore
cd ../../
