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

# Glide
echo "Getting glide if not found"
go get github.com/Masterminds/glide

# First load glide deps
echo "Checking snap root for deps"
glide install
