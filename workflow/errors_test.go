// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workflow

import (
	"errors"
	"strings"
	"testing"
)

func TestSentinels_NonNilAndDistinct(t *testing.T) {
	sentinels := map[string]error{
		"ErrNodeFailed":              ErrNodeFailed,
		"ErrNodeInterrupted":         ErrNodeInterrupted,
		"ErrInvalidRunNodeContext":   ErrInvalidRunNodeContext,
		"ErrInvalidRunID":            ErrInvalidRunID,
		"ErrParallelHITLUnsupported": ErrParallelHITLUnsupported,
	}

	for name, err := range sentinels {
		t.Run(name+"/non_nil", func(t *testing.T) {
			if err == nil {
				t.Errorf("%s is nil; want non-nil sentinel", name)
			}
		})
		t.Run(name+"/errors_is_matches_self", func(t *testing.T) {
			if !errors.Is(err, err) {
				t.Errorf("errors.Is(%s, %s) = false; want true", name, name)
			}
		})
		t.Run(name+"/has_workflow_prefix", func(t *testing.T) {
			if !strings.HasPrefix(err.Error(), "workflow:") {
				t.Errorf("%s.Error() = %q; want prefix \"workflow:\"", name, err.Error())
			}
		})
	}

	names := make([]string, 0, len(sentinels))
	for n := range sentinels {
		names = append(names, n)
	}
	for i, a := range names {
		for _, b := range names[i+1:] {
			if errors.Is(sentinels[a], sentinels[b]) {
				t.Errorf("errors.Is(%s, %s) = true; sentinels must be distinct", a, b)
			}
		}
	}
}

func TestNodeRunError_Unwrap(t *testing.T) {
	t.Run("returns Cause", func(t *testing.T) {
		nre := &NodeRunError{Cause: ErrNodeFailed}
		if got := nre.Unwrap(); got != ErrNodeFailed {
			t.Errorf("Unwrap() = %v, want %v", got, ErrNodeFailed)
		}
	})
	t.Run("nil receiver returns nil", func(t *testing.T) {
		var nre *NodeRunError
		if got := nre.Unwrap(); got != nil {
			t.Errorf("(*NodeRunError)(nil).Unwrap() = %v, want nil", got)
		}
	})
	t.Run("nil Cause returns nil", func(t *testing.T) {
		nre := &NodeRunError{}
		if got := nre.Unwrap(); got != nil {
			t.Errorf("Unwrap() with nil Cause = %v, want nil", got)
		}
	})
}

func TestNodeRunError_ErrorsIs(t *testing.T) {
	nre := &NodeRunError{
		ChildName: "fixer",
		ChildPath: "code_workflow/fixer@2",
		RunID:     "2",
		Cause:     ErrNodeFailed,
	}
	if !errors.Is(nre, ErrNodeFailed) {
		t.Errorf("errors.Is(NodeRunError{Cause: ErrNodeFailed}, ErrNodeFailed) = false; want true")
	}
	if errors.Is(nre, ErrNodeInterrupted) {
		t.Errorf("errors.Is(NodeRunError{Cause: ErrNodeFailed}, ErrNodeInterrupted) = true; want false")
	}
}

func TestNodeRunError_ErrorsAs(t *testing.T) {
	original := &NodeRunError{
		ChildName: "lint_check",
		ChildPath: "code_workflow/lint_check@3",
		RunID:     "3",
		Cause:     ErrNodeInterrupted,
	}
	// Wrap once more to simulate propagation through extra layers.
	wrapped := errors.Join(errors.New("upstream context"), original)

	var got *NodeRunError
	if !errors.As(wrapped, &got) {
		t.Fatalf("errors.As(wrapped, &*NodeRunError) = false; want true")
	}
	if got != original {
		t.Errorf("errors.As recovered different *NodeRunError; got %+v, want %+v", got, original)
	}
}

func TestNodeRunError_ErrorFormat(t *testing.T) {
	tests := []struct {
		name string
		nre  *NodeRunError
		want string
	}{
		{
			name: "happy path",
			nre: &NodeRunError{
				ChildPath: "code_workflow/fixer@2",
				Cause:     ErrNodeFailed,
			},
			want: "workflow: dynamic child code_workflow/fixer@2: workflow: dynamic child failed",
		},
		{
			name: "falls back to ChildName when ChildPath empty",
			nre: &NodeRunError{
				ChildName: "fixer",
				Cause:     ErrNodeInterrupted,
			},
			want: "workflow: dynamic child fixer: workflow: dynamic child interrupted",
		},
		{
			name: "nil cause",
			nre: &NodeRunError{
				ChildPath: "x/y@1",
			},
			want: "workflow: dynamic child x/y@1: <nil cause>",
		},
		{
			name: "nil receiver",
			nre:  nil,
			want: "<nil>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.nre.Error(); got != tc.want {
				t.Errorf("Error()\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// Compile-time: *NodeRunError implements error.
var _ error = (*NodeRunError)(nil)
