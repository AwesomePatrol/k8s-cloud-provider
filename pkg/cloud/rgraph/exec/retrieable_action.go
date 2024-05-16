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

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
)

// RetryProvider decides if action should be retired on error
type RetryProvider interface {
	IsRetriable(error) bool
}

// retriableAction implements Action
var _ Action = (*retriableAction)(nil)

// retriableAction is a decorator to the action adding retry mechanism
type retriableAction struct {
	a             Action
	retryProvider RetryProvider
}

// NewRetriableAction decorates Action with retry provider
func NewRetriableAction(a Action, rp RetryProvider) Action {
	return &retriableAction{a, rp}
}

// CanRun indicate if all preconditions to run Action are met and action can be
// executed
func (ra *retriableAction) CanRun() bool {
	return ra.a.CanRun()
}

// Signal notifies parents that Action was executed
func (ra *retriableAction) Signal(e Event) bool {
	return ra.a.Signal(e)
}

// Run executes Action. If error is returned retry provider checks if Action
// should be rerun
func (ra *retriableAction) Run(ctx context.Context, c cloud.Cloud) (EventList, error) {
	var err error
	var events EventList
	for run := true; run; run = ra.retryProvider.IsRetriable(err) && ctx.Err() == nil {
		events, err = ra.a.Run(ctx, c)
		if err == nil {
			return events, err
		}
	}

	return events, err
}

// DryRun returns post action events
func (ra *retriableAction) DryRun() EventList {
	return ra.a.DryRun()
}

// String wraps Action name with retry information
func (ra *retriableAction) String() string {
	return ra.a.String() + " with retry"
}

// PendingEvents returns a list of events the Action is waiting to complete
func (ra *retriableAction) PendingEvents() EventList {
	return ra.a.PendingEvents()
}

// Metadata returns wrapped Action's metadata
func (ra *retriableAction) Metadata() *ActionMetadata {
	return ra.a.Metadata()
}
