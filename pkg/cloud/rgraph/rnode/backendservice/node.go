/*
Copyright 2023 Google LLC

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

package backendservice

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func nodeErr(s string, args ...any) error { return fmt.Errorf("backendService: "+s, args...) }

type backendServiceNode struct {
	rnode.NodeBase
	resource BackendService
}

var _ rnode.Node = (*backendServiceNode)(nil)

func (n *backendServiceNode) Resource() rnode.UntypedResource { return n.resource }

func (n *backendServiceNode) Diff(gotNode rnode.Node) (*rnode.PlanDetails, error) {
	got, ok := gotNode.(*backendServiceNode)
	if !ok {
		return nil, fmt.Errorf("BackendServiceNode: invalid type to Diff: %T", gotNode)
	}
	diff, err := got.resource.Diff(n.resource)
	if err != nil {
		return nil, fmt.Errorf("BackendServiceNode: Diff %w", err)
	}

	if !diff.HasDiff() {
		return &rnode.PlanDetails{
			Operation: rnode.OpNothing,
			Why:       "No diff between got and want",
		}, nil
	}

	var (
		needsRecreate bool
		details       []string
	)

	planRecreate := func(s string, args ...any) {
		details = append(details, fmt.Sprintf(s, args...))
		needsRecreate = true
	}

	for _, delta := range diff.Items {
		switch {
		case delta.Path.Equal(api.Path{}.Pointer().Field("LoadBalancingScheme")):
			planRecreate("LoadBalancingScheme change: '%v' -> '%v'", delta.A, delta.B)
			//default:
			//	planRecreate("%s change: '%v' -> '%v'", delta.Path, delta.A, delta.B)
		}
	}

	if needsRecreate {
		return &rnode.PlanDetails{
			Operation: rnode.OpRecreate,
			Why:       "BackendService needs to be recreated: " + strings.Join(details, ", "),
			Diff:      diff,
		}, nil
	}
	return &rnode.PlanDetails{
		Operation: rnode.OpUpdate,
		Why:       fmt.Sprintf("update in place (changed=TODO)"),
		Diff:      diff,
	}, nil
}

func (n *backendServiceNode) Actions(got rnode.Node) ([]exec.Action, error) {
	op := n.Plan().Op()

	switch op {
	case rnode.OpCreate:
		return rnode.CreateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, n, n.resource)

	case rnode.OpDelete:
		return rnode.DeleteActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, got, n)

	case rnode.OpNothing:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil

	case rnode.OpRecreate:
		return rnode.RecreateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, got, n, n.resource)

	case rnode.OpUpdate:
		return rnode.UpdateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, got, n, n.resource)
	}

	return nil, fmt.Errorf("BackendServiceNode: invalid plan op %s", op)
}

func (n *backendServiceNode) Builder() rnode.Builder {
	b := &builder{}
	b.Init(n.ID(), n.State(), n.Ownership(), n.resource)
	return b
}
