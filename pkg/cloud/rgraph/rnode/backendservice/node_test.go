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
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/networkendpointgroup"
	"github.com/google/go-cmp/cmp"
	"github.com/kr/pretty"
	"google.golang.org/api/compute/v1"
)

func TestNodeBuilder(t *testing.T) {
	id := ID("proj", meta.GlobalKey("bs"))
	b := NewBuilder(id)
	b.SetOwnership(rnode.OwnershipExternal)
	b.SetState(rnode.NodeDoesNotExist)
	n, err := b.Build()
	if err != nil {
		t.Fatalf("Build() = %v", err)
	}
	b2 := n.Builder()
	type result struct {
		O rnode.OwnershipStatus
		S rnode.NodeState
	}
	if diff := cmp.Diff(
		result{O: b2.Ownership(), S: b2.State()},
		result{O: rnode.OwnershipExternal, S: rnode.NodeDoesNotExist},
	); diff != "" {
		t.Fatalf("Diff() -got,+want: %s", diff)
	}
}

func TestDiffAndActions(t *testing.T) {
	id := ID("proj", meta.GlobalKey("bs"))
	negID := networkendpointgroup.ID("proj", meta.GlobalKey("neg"))
	negID2 := networkendpointgroup.ID("proj", meta.GlobalKey("neg2"))

	const (
		ignoreAccessErr = 1 << iota
	)

	makeBS := func(f func(x *compute.BackendService), flags int) BackendService {
		t.Helper()

		fr := NewMutableBackendService(id.ProjectID, id.Key)
		fr.Access(func(x *compute.BackendService) {
			x.Name = "bs"
		})
		if f != nil {
			err := fr.Access(f)
			if err != nil && flags&ignoreAccessErr == 0 {
				t.Fatalf("Access() = %v, want nil", err)
			}
		}
		r, err := fr.Freeze()
		if err != nil {
			t.Fatalf("fr.Freeze() = %v, want nil", err)
		}
		return r
	}

	baseFields := func(x *compute.BackendService) {
		x.Backends = []*compute.Backend{{Group: negID.SelfLink(meta.VersionGA)}}
		x.NullFields = []string{}
		x.ForceSendFields = []string{
			"AffinityCookieTtlSec",
			"CdnPolicy",
			"CircuitBreakers",
			"CompressionMode",
			"ConnectionDraining",
			"ConnectionTrackingPolicy",
			"ConsistentHash",
			"CustomRequestHeaders",
			"CustomResponseHeaders",
			"Description",
			"EnableCDN",
			"FailoverPolicy",
			"Iap",
			"LoadBalancingScheme",
			"LocalityLbPolicies",
			"LocalityLbPolicy",
			"LogConfig",
			"MaxStreamDuration",
			"Network",
			"OutlierDetection",
			"Port",
			"PortName",
			"Protocol",
			"SecuritySettings",
			"ServiceBindings",
			"SessionAffinity",
			"Subsetting",
			"TimeoutSec",
			//"Backends",
		}
	}

	for _, tc := range []struct {
		name     string
		bsw, bsg BackendService

		wantDiff       bool
		wantOp         rnode.Operation
		wantErr        bool
		wantActionsErr bool
		wantActions    []string
	}{
		{
			name: "no diff",
			bsw: makeBS(func(x *compute.BackendService) {
				baseFields(x)
			}, 0),
			bsg: makeBS(func(x *compute.BackendService) {
				baseFields(x)
			}, ignoreAccessErr),
			wantOp: rnode.OpNothing,
			wantActions: []string{
				"EventAction([Exists(compute/backendService:proj/bs)])",
			},
		},
		{
			name: "update .Target",
			bsw: makeBS(func(x *compute.BackendService) {
				baseFields(x)
			}, 0),
			bsg: makeBS(func(x *compute.BackendService) {
				baseFields(x)
				x.Backends = []*compute.Backend{{Group: negID2.SelfLink(meta.VersionGA)}}
			}, ignoreAccessErr),
			wantDiff: true,
			wantOp:   rnode.OpUpdate,
			wantActions: []string{
				"EventAction([Exists(compute/backendService:proj/bs)])",
				"BackendServiceUpdateAction(compute/backendService:proj/bs)",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bg := NewBuilderWithResource(tc.bsg)
			bw := NewBuilderWithResource(tc.bsw)

			ng, err := bg.Build()
			if err != nil {
				t.Fatalf("bg.Build() = %v, want nil", err)
			}
			nw, err := bw.Build()
			if err != nil {
				t.Fatalf("bw.Build() = %v, want nil", err)
			}

			pd, err := ng.Diff(nw)
			t.Logf("Diff() = %v; %s", err, pretty.Sprint(pd))
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("")
			}
			if gotDiff := pd.Diff != nil && pd.Diff.HasDiff(); gotDiff != tc.wantDiff {
				t.Errorf("gotDiff = %t, want %t", gotDiff, tc.wantDiff)
			}
			if gotOp := pd.Operation; gotOp != tc.wantOp {
				t.Errorf("gotOp = %s, want %s", gotOp, tc.wantOp)
			}
			// Set the plan to be the same as given by the diff.
			nw.Plan().Set(rnode.PlanDetails{
				Operation: pd.Operation,
				Diff:      pd.Diff,
			})
			actions, err := nw.Actions(ng)
			if gotActionsErr := err != nil; gotActionsErr != tc.wantActionsErr {
				t.Fatalf("Actions() = %v, want nil", err)
			}
			var strActions []string
			for _, act := range actions {
				strActions = append(strActions, fmt.Sprint(act))
			}
			if diff := cmp.Diff(strActions, tc.wantActions); diff != "" {
				t.Errorf("Diff(actions) -got,+want: %s", diff)
			}
		})
	}
}
