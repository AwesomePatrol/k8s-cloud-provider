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
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/networkendpointgroup"
	"github.com/google/go-cmp/cmp"
	compute "google.golang.org/api/compute/v1"
)

func TestBuilderSetResource(t *testing.T) {
	id := ID("proj", meta.GlobalKey("bs"))

	newR := func() rnode.UntypedResource {
		mr := NewMutableBackendService(id.ProjectID, id.Key)
		r, _ := mr.Freeze()
		return r
	}

	for _, tc := range []struct {
		name    string
		r       rnode.UntypedResource
		wantErr bool
	}{
		{
			name: "ok",
			r:    newR(),
		},
		{
			name:    "wrong type",
			r:       fake.Fake(nil), // this will fail to cast.
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder(id)
			err := b.SetResource(tc.r)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("SetResource() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
		})
	}
}

func TestOutRefs(t *testing.T) {
	id := ID("proj", meta.GlobalKey("fr"))
	negID := networkendpointgroup.ID("proj", meta.GlobalKey("neg"))

	for _, tc := range []struct {
		name string
		f    func(*compute.BackendService)

		wantErr bool
		want    []rnode.ResourceRef
	}{
		{
			name: "backends",
			f: func(x *compute.BackendService) {
				x.Backends = []*compute.Backend{{Group: negID.SelfLink(meta.VersionGA)}}
			},
			want: []rnode.ResourceRef{
				{From: id, To: negID, Path: api.Path{}.Field("Backends").Index(0).Field("Group")},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := NewMutableBackendService(id.ProjectID, id.Key)
			mr.Access(tc.f)
			r, _ := mr.Freeze()
			b := NewBuilderWithResource(r)

			got, err := b.OutRefs()
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("XXX")
			} else if gotErr {
				return
			}
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("OutRefs diff = -got,+want: %s", diff)
			}
		})
	}
}
