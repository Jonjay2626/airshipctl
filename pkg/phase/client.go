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

package phase

import (
	"io"
	"io/ioutil"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"opendev.org/airship/airshipctl/pkg/api/v1alpha1"
	cctlclient "opendev.org/airship/airshipctl/pkg/clusterctl/client"
	"opendev.org/airship/airshipctl/pkg/document"
	"opendev.org/airship/airshipctl/pkg/events"
	"opendev.org/airship/airshipctl/pkg/k8s/kubeconfig"
	"opendev.org/airship/airshipctl/pkg/k8s/utils"
	"opendev.org/airship/airshipctl/pkg/log"
	"opendev.org/airship/airshipctl/pkg/phase/errors"
	"opendev.org/airship/airshipctl/pkg/phase/executors"
	executorerrors "opendev.org/airship/airshipctl/pkg/phase/executors/errors"
	"opendev.org/airship/airshipctl/pkg/phase/ifc"
)

// ExecutorRegistry returns map with executor factories
type ExecutorRegistry func() map[schema.GroupVersionKind]ifc.ExecutorFactory

// DefaultExecutorRegistry returns map with executor factories
func DefaultExecutorRegistry() map[schema.GroupVersionKind]ifc.ExecutorFactory {
	execMap := make(map[schema.GroupVersionKind]ifc.ExecutorFactory)

	for _, execName := range []string{executors.Clusterctl, executors.KubernetesApply,
		executors.GenericContainer, executors.Ephemeral, executors.BMHManager} {
		if err := executors.RegisterExecutor(execName, execMap); err != nil {
			log.Fatal(executorerrors.ErrExecutorRegistration{ExecutorName: execName, Err: err})
		}
	}
	return execMap
}

var _ ifc.Phase = &phase{}

// Phase implements phase interface
type phase struct {
	helper    ifc.Helper
	apiObj    *v1alpha1.Phase
	registry  ExecutorRegistry
	processor events.EventProcessor
}

// Executor returns executor interface associated with the phase
func (p *phase) Executor() (ifc.Executor, error) {
	executorDoc, err := p.helper.ExecutorDoc(ifc.ID{Name: p.apiObj.Name, Namespace: p.apiObj.Namespace})
	if err != nil {
		return nil, err
	}

	var bundleFactory document.BundleFactoryFunc = func() (document.Bundle, error) {
		docRoot, bundleFactoryFuncErr := p.DocumentRoot()
		if bundleFactoryFuncErr != nil {
			return nil, bundleFactoryFuncErr
		}
		return document.NewBundleByPath(docRoot)
	}

	refGVK := p.apiObj.Config.ExecutorRef.GroupVersionKind()
	// Look for executor factory defined in registry
	executorFactory, found := p.registry()[refGVK]
	if !found {
		return nil, executorerrors.ErrExecutorNotFound{GVK: refGVK}
	}

	cMap, err := p.helper.ClusterMap()
	if err != nil {
		return nil, err
	}

	cctlClient, err := cctlclient.NewClient(
		p.helper.PhaseBundleRoot(),
		log.DebugEnabled(),
		v1alpha1.DefaultClusterctl())
	if err != nil {
		return nil, err
	}

	kubeconf := kubeconfig.NewBuilder().
		WithBundle(p.helper.PhaseConfigBundle()).
		WithClusterMap(cMap).
		WithTempRoot(p.helper.WorkDir()).
		WithClusterctlClient(cctlClient).
		Build()

	return executorFactory(
		ifc.ExecutorConfig{
			ClusterMap:        cMap,
			BundleFactory:     bundleFactory,
			PhaseName:         p.apiObj.Name,
			KubeConfig:        kubeconf,
			ExecutorDocument:  executorDoc,
			ClusterName:       p.apiObj.ClusterName,
			PhaseConfigBundle: p.helper.PhaseConfigBundle(),
			SinkBasePath:      p.helper.PhaseEntryPointBasePath(),
			TargetPath:        p.helper.TargetPath(),
			Inventory:         p.helper.Inventory(),
		})
}

// Run runs the phase via executor
func (p *phase) Run(ro ifc.RunOptions) error {
	defer p.processor.Close()
	executor, err := p.Executor()
	if err != nil {
		return err
	}
	ch := make(chan events.Event)

	go func() {
		executor.Run(ch, ro)
	}()
	return p.processor.Process(ch)
}

// Validate makes sure that phase is properly configured
func (p *phase) Validate() error {
	// Check that we can render documents supplied to phase
	err := p.Render(ioutil.Discard, false, ifc.RenderOptions{})
	if err != nil {
		return err
	}

	// Check that executor if properly configured
	executor, err := p.Executor()
	if err != nil {
		return err
	}
	return executor.Validate()
}

// Render executor documents
func (p *phase) Render(w io.Writer, executorRender bool, options ifc.RenderOptions) error {
	if executorRender {
		executor, err := p.Executor()
		if err != nil {
			return err
		}
		return executor.Render(w, options)
	}

	root, err := p.DocumentRoot()
	if err != nil {
		return err
	}

	bundle, err := document.NewBundleByPath(root)
	if err != nil {
		return err
	}

	rendered, err := bundle.SelectBundle(options.FilterSelector)
	if err != nil {
		return err
	}
	return rendered.Write(w)
}

// Status returns the status of the given phase
func (p *phase) Status() (ifc.PhaseStatus, error) {
	executor, err := p.Executor()
	if err != nil {
		return ifc.PhaseStatus{}, err
	}

	sts, err := executor.Status()
	if err != nil {
		return ifc.PhaseStatus{}, err
	}

	return ifc.PhaseStatus{ExecutorStatus: sts}, err
}

// DocumentRoot root that holds all the documents associated with the phase
func (p *phase) DocumentRoot() (string, error) {
	relativePath := p.apiObj.Config.DocumentEntryPoint
	if relativePath == "" {
		return "", errors.ErrDocumentEntrypointNotDefined{
			PhaseName:      p.apiObj.Name,
			PhaseNamespace: p.apiObj.Namespace,
		}
	}
	return filepath.Join(p.helper.PhaseEntryPointBasePath(), relativePath), nil
}

// Details returns description of the phase
// TODO implement this: add details field to api.Phase and method to executor and combine them here
// to give a clear understanding to user of what this phase is about
func (p *phase) Details() (string, error) {
	return "", nil
}

var _ ifc.Plan = &plan{}

type plan struct {
	helper      ifc.Helper
	apiObj      *v1alpha1.PhasePlan
	phaseClient ifc.Client
}

// Validate makes sure that phase plan is properly configured
func (p *plan) Validate() error {
	for _, step := range p.apiObj.Phases {
		phaseRunner, err := p.phaseClient.PhaseByID(ifc.ID{Name: step.Name})
		if err != nil {
			return err
		}
		if err = phaseRunner.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Run function executes Run method for each phase
func (p *plan) Run(ro ifc.RunOptions) error {
	for _, step := range p.apiObj.Phases {
		phaseRunner, err := p.phaseClient.PhaseByID(ifc.ID{Name: step.Name})
		if err != nil {
			return err
		}

		log.Printf("executing phase: %s\n", step)
		if err = phaseRunner.Run(ro); err != nil {
			return err
		}
	}
	return nil
}

var _ ifc.Client = &client{}

type client struct {
	ifc.Helper

	registry      ExecutorRegistry
	processorFunc ProcessorFunc
}

// ProcessorFunc that returns processor interface
type ProcessorFunc func() events.EventProcessor

// Option allows to add various options to a phase
type Option func(*client)

// InjectRegistry is an option that allows to inject executor registry into phase client
func InjectRegistry(registry ExecutorRegistry) Option {
	return func(c *client) {
		c.registry = registry
	}
}

// NewClient returns implementation of phase Client interface
func NewClient(helper ifc.Helper, opts ...Option) ifc.Client {
	c := &client{Helper: helper}
	for _, opt := range opts {
		opt(c)
	}
	if c.registry == nil {
		c.registry = DefaultExecutorRegistry
	}
	if c.processorFunc == nil {
		c.processorFunc = defaultProcessor
	}
	return c
}

func (c *client) PhaseByID(id ifc.ID) (ifc.Phase, error) {
	phaseObj, err := c.Phase(id)
	if err != nil {
		return nil, err
	}

	phase := &phase{
		apiObj:    phaseObj,
		helper:    c.Helper,
		processor: c.processorFunc(),
		registry:  c.registry,
	}
	return phase, nil
}

func (c *client) PlanByID(id ifc.ID) (ifc.Plan, error) {
	planObj, err := c.Plan(id)
	if err != nil {
		return nil, err
	}
	return &plan{
		apiObj:      planObj,
		helper:      c.Helper,
		phaseClient: c,
	}, nil
}

func (c *client) PhaseByAPIObj(phaseObj *v1alpha1.Phase) (ifc.Phase, error) {
	phase := &phase{
		apiObj:    phaseObj,
		helper:    c.Helper,
		processor: c.processorFunc(),
		registry:  c.registry,
	}
	return phase, nil
}

func defaultProcessor() events.EventProcessor {
	return events.NewDefaultProcessor(utils.Streams())
}
