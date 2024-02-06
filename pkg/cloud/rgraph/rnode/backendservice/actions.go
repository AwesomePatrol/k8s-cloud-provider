/*
Copyright 2024 Google LLC

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
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

type backendServiceUpdateAction struct {
	exec.ActionBase

	id   *cloud.ResourceID
	want *backendServiceNode
}

func (act *backendServiceUpdateAction) Run(ctx context.Context, cl cloud.Cloud) (exec.EventList, error) {
	res, err := act.want.resource.ToGA()
	if err != nil {
		return nil, fmt.Errorf("backendServiceUpdateAction Run(%s): ToGA: %w", act.id, err)
	}
	// TODO: project routing.
	switch act.id.Key.Type() {
	case meta.Global:
		err := cl.BackendServices().Update(ctx, act.id.Key, res)
		if err != nil {
			return nil, fmt.Errorf("backendServiceUpdateAction Run(%s): Update: %w", act.id, err)
		}
	case meta.Regional:
		err := cl.RegionBackendServices().Update(ctx, act.id.Key, res)
		if err != nil {
			return nil, fmt.Errorf("backendServiceUpdateAction Run(%s): Update: %w", act.id, err)
		}
	default:
		return nil, fmt.Errorf("backendServiceUpdateAction Run(%s): invalid key type", act.id)
	}

	// TODO: manage references to backends/groups
	return nil, nil
}

func (act *backendServiceUpdateAction) DryRun() exec.EventList {
	return nil
}

func (act *backendServiceUpdateAction) String() string {
	return fmt.Sprintf("BackendServiceUpdateAction(%s)", act.id)
}

func (act *backendServiceUpdateAction) Metadata() *exec.ActionMetadata {
	return &exec.ActionMetadata{
		Name:    fmt.Sprintf("BackendServiceUpdateAction(%s)", act.id),
		Type:    exec.ActionTypeUpdate,
		Summary: fmt.Sprintf("Update %s", act.id),
	}
}
