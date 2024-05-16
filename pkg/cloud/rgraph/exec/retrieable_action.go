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
	Action
	retryProvider RetryProvider
}

// NewRetriableAction decorates Action with retry provider
func NewRetriableAction(a Action, rp RetryProvider) Action {
	return &retriableAction{a, rp}
}

// Run executes Action. If error is returned retry provider checks if Action
// should be rerun
func (ra *retriableAction) Run(ctx context.Context, c cloud.Cloud) (EventList, error) {
	var err error
	var events EventList
	for run := true; run; run = ra.retryProvider.IsRetriable(err) && ctx.Err() == nil {
		events, err = ra.Action.Run(ctx, c)
		if err == nil {
			return events, err
		}
	}

	return events, err
}

// String wraps Action name with retry information
func (ra *retriableAction) String() string {
	return ra.Action.String() + " with retry"
}
