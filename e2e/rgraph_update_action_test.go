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
package e2e

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/testing/ez"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/plan"
	"google.golang.org/api/compute/v1"
)

func TestHcUpdate(t *testing.T) {
	ctx := context.Background()
	hc := defaultHTTPHC("hc-e2e-name")

	newG := hcGraph(&hc)
	execResult := executeRGraph(t, newG)
	t.Logf(" === Executors Results === \n %+v", execResult)
	key := meta.GlobalKey(hc.Name)
	got, err := theCloud.HealthChecks().Get(ctx, key)
	if err != nil {
		t.Fatalf("theCloud.HealthChecks().Get(_, %s) = %v, want nil", meta.GlobalKey(hc.Name).String(), err)
	}

	t.Cleanup(func() {
		err := theCloud.HealthChecks().Delete(ctx, key)
		if err != nil {
			t.Logf("delete health check: %v", err)
		}
	})
	if !cmp(got, &hc) {
		t.Fatalf("HealthCheck objects are not equal got: %+v, want %+v", got, hc)
	}

	// update health check
	hc.HttpHealthCheck.Port = 123
	hc.CheckIntervalSec = 60

	updatedG := hcGraph(&hc)
	execResult = executeRGraph(t, updatedG)
	t.Logf(" === Executors Results === \n %+v", execResult)

	got, err = theCloud.HealthChecks().Get(ctx, key)
	if err != nil {
		t.Fatalf("theCloud.HealthChecks().Get(_, %s) = %v, want nil", meta.GlobalKey(hc.Name).String(), err)
	}
	if !cmp(got, &hc) {
		t.Fatalf("HealthCheck objects are not equal got: %+v, want %+v", got, hc)
	}
}

func TestHcUpdateWithBackendService(t *testing.T) {
	ctx := context.Background()
	hc := defaultHTTPHC("hc-e2e-bs")
	bs := defaultBS("bs-e2e", &hc)
	newG := bsGraph(&hc, bs)
	execResult := executeRGraph(t, newG)
	t.Logf(" === Executors Results === \n %+v", execResult)
	hcKey := meta.GlobalKey(hc.Name)
	gotHC, err := theCloud.HealthChecks().Get(ctx, hcKey)
	if err != nil {
		t.Fatalf("theCloud.HealthChecks().Get(_, %s) = %v, want nil", hcKey.String(), err)
	}
	bsKey := meta.GlobalKey(bs.Name)
	gotBS, err := theCloud.BackendServices().Get(ctx, bsKey)
	if err != nil {
		t.Fatalf("theCloud.HealthChecks().Get(_, %s) = %v, want nil", bsKey.String(), err)
	}

	t.Cleanup(func() {
		err := theCloud.HealthChecks().Delete(ctx, hcKey)
		if err != nil {
			t.Logf("delete health check: %v", err)
		}
		err = theCloud.BackendServices().Delete(ctx, bsKey)
		if err != nil {
			t.Logf("delete backend service: %v", err)
		}
	})
	if !cmp(gotHC, &hc) {
		t.Fatalf("HealthCheck objects are not equal got: %+v, want %+v", gotHC, hc)
	}
	if len(gotBS.HealthChecks) == 0 || gotBS.HealthChecks[0] != hc.SelfLink {
		t.Fatalf("BackendService %s does not have health check %s set", bsKey.String(), hc.SelfLink)
	}
	// update health check
	hc.HttpHealthCheck.Port = 123
	hc.CheckIntervalSec = 60

	updatedG := hcGraph(&hc)
	execResult = executeRGraph(t, updatedG)
	t.Logf(" === Executors Results === \n %+v", execResult)

	gotHC, err = theCloud.HealthChecks().Get(ctx, hcKey)
	if err != nil {
		t.Fatalf("theCloud.HealthChecks().Get(_, %s) = %v, want nil", hcKey.String(), err)
	}
	if !cmp(gotHC, &hc) {
		t.Fatalf("HealthCheck objects are not equal got: %+v, want %+v", gotHC, hc)
	}
}

func defaultHTTPHC(name string) compute.HealthCheck {
	return compute.HealthCheck{
		Name:             name,
		CheckIntervalSec: 10,
		HttpHealthCheck: &compute.HTTPHealthCheck{
			Port:     80,
			PortName: "http",
		},
		Type: "HTTP",
	}
}

func defaultBS(name string, hc *compute.HealthCheck) *compute.BackendService {
	return &compute.BackendService{
		Name:                name,
		HealthChecks:        []string{hc.SelfLink},
		Backends:            []*compute.Backend{},
		LoadBalancingScheme: "INTERNAL_SELF_MANAGED",
		Protocol:            "TCP",
	}
}

func hcGraph(hc *compute.HealthCheck) *rgraph.Graph {
	ezg := ez.Graph{
		Nodes: []ez.Node{
			{Name: hc.Name,
				SetupFunc: func(x *compute.HealthCheck) {
					*x = *hc
				},
			},
		},
		Project: "katarzynalach-gke-dev",
	}
	return ezg.Builder().MustBuild()
}
func bsGraph(hc *compute.HealthCheck, bs *compute.BackendService) *rgraph.Graph {
	ezg := ez.Graph{
		Nodes: []ez.Node{
			{
				Name: hc.Name,
				SetupFunc: func(x *compute.HealthCheck) {
					*x = *hc
				},
			},
			{
				Name: bs.Name,
				SetupFunc: func(x *compute.BackendService) {
					*x = *bs
				},
				Refs: []ez.Ref{{Field: "Healthchecks", To: hc.Name}}},
		},
		Project: "katarzynalach-gke-dev",
	}
	return ezg.Builder().MustBuild()
}

func executeRGraph(t *testing.T, g *rgraph.Graph) *exec.Result {
	t.Helper()
	result, err := plan.Do(context.Background(), theCloud, g)
	t.Logf("====== \n planning returned results: %+v", result)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}

	ex, err := exec.NewSerialExecutor(result.Actions, exec.DryRunOption(false))
	if err != nil {
		t.Errorf("NewSerialExecutor() = %v, want nil", err)
		return nil
	}

	execResult, err := ex.Run(context.Background(), theCloud)
	if err != nil {
		t.Errorf("ex.Run() = %v, want nil", err)
		return nil
	}
	return execResult
}

func cmp(hc1, hc2 *compute.HealthCheck) bool {
	if hc1.Name != hc2.Name {
		return false
	}
	if hc1.Type != hc2.Type {
		return false
	}
	if hc1.CheckIntervalSec != hc2.CheckIntervalSec {
		return false
	}
	if hc1.HttpHealthCheck == nil || hc2.HttpHealthCheck == nil {
		return false
	}
	if hc1.HttpHealthCheck.Port != hc2.HttpHealthCheck.Port {
		return false
	}
	if hc1.HttpHealthCheck.PortName != hc2.HttpHealthCheck.PortName {
		return false
	}
	return true
}
