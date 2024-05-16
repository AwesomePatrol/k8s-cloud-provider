/*
Copyright 2024 Google LLC

You may obtain a copy of the License at

https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package exec

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
)

// fakeAction will return error for n actions defined in errorRunThreshold,
// runCtr counts all action executions.
// errorRunThreshold set to -1 means that Action should always return error.
type fakeAction struct {
	runCtr            int
	errorRunThreshold int
}

func (fa *fakeAction) CanRun() bool {
	return true
}

func (fa *fakeAction) Signal(e Event) bool {
	return false
}

func (fa *fakeAction) Run(ctx context.Context, c cloud.Cloud) (EventList, error) {
	fa.runCtr++
	if fa.errorRunThreshold > fa.runCtr {
		return EventList{}, fmt.Errorf("Action in error")
	}
	return EventList{}, nil
}

func (fa *fakeAction) DryRun() EventList {
	return EventList{}
}

func (fa *fakeAction) String() string {
	return "fakeAction"
}

func (fa *fakeAction) PendingEvents() EventList {
	return EventList{}
}

func (fa *fakeAction) Metadata() *ActionMetadata {
	return &ActionMetadata{
		Name: "fakeAction",
	}
}

// fakeRetryProvider implements RetryProvider.
// ctr - counts all retry provider calls
// shouldRetry - tells if Action should be rerun
type fakeRetryProvider struct {
	ctr         int
	shouldRetry bool
}

// IsRetriable returns info if action should be rerun. Every call to this
// function increments counter.
func (frp *fakeRetryProvider) IsRetriable(error) bool {
	frp.ctr++
	return frp.shouldRetry
}

func TestRetriableAction(t *testing.T) {
	for _, tc := range []struct {
		name             string
		shouldRetry      bool
		wantError        bool
		wantRunThreshold int
		wantRetries      int
		wantRun          int
	}{
		{

			name:             "should not retry",
			shouldRetry:      false,
			wantError:        true,
			wantRunThreshold: 5,
			wantRetries:      1,
			wantRun:          1,
		},
		{

			name:             "should retry",
			shouldRetry:      true,
			wantError:        false,
			wantRunThreshold: 5,
			wantRetries:      4,
			wantRun:          5,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fa := &fakeAction{errorRunThreshold: tc.wantRunThreshold}
			frp := &fakeRetryProvider{shouldRetry: tc.shouldRetry}
			ra := NewRetriableAction(fa, frp)
			_, err := ra.Run(context.Background(), nil)
			gotErr := err != nil
			if gotErr != tc.wantError {
				t.Fatalf("ra.Run(context.Background(), nil) = %v, gotErr: %v, wantErr : %v", err, gotErr, tc.wantError)
			}

			if fa.runCtr != tc.wantRun {
				t.Errorf("action run mismatch: got %v, want 4", fa.runCtr)
			}

			if frp.ctr != tc.wantRetries {
				t.Errorf("retires mismatch: got %v, want %v", frp.ctr, tc.wantRetries)
			}
		})
	}
}

func TestRetriableActionWithContextCancel(t *testing.T) {
	fa := &fakeAction{errorRunThreshold: -1}
	frp := &fakeRetryProvider{shouldRetry: true}
	ra := NewRetriableAction(fa, frp)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_, err := ra.Run(ctx, nil)
	cancel()

	if err != nil {
		t.Fatalf("ra.Run(context.Background(), nil) = %v, want nil", err)
	}

	if fa.runCtr > 1 {
		t.Errorf("action run mismatch: got %v, want 4", fa.runCtr)
	}

	if frp.ctr > 1 {
		t.Errorf("retires mismatch: got %v, want 1", frp.ctr)
	}
}
