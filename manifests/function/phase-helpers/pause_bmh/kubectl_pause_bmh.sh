#!/bin/sh

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

set -xe

echo "Annotating BMH objects with a pause label in cluster with kubectl context ${KCTL_CONTEXT}" 1>&2
kubectl annotate \
  --context $KCTL_CONTEXT \
  --namespace $CLUSTER_NAMESPACE \
  --overwrite \
  -f $RENDERED_BUNDLE_PATH baremetalhost.metal3.io/paused=true 1>&2
