/*
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package utils

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// ClientOption is a function that allows to modify ConfigFlags object which is used to create Client
type ClientOption func(*genericclioptions.ConfigFlags)

// SetTimeout sets Timeout option in ConfigFlags object
func SetTimeout(timeout string) ClientOption {
	return func(co *genericclioptions.ConfigFlags) {
		*co.Timeout = timeout
	}
}

// FactoryFromKubeConfig returns a factory with the
// default Kubernetes resources for the given kube config path and context
func FactoryFromKubeConfig(path, context string, opts ...ClientOption) cmdutil.Factory {
	kf := genericclioptions.NewConfigFlags(false)
	kf.KubeConfig = &path
	kf.Context = &context
	for _, o := range opts {
		o(kf)
	}

	return cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(kf))
}
