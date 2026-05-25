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
	"context"
	"testing"
)

func TestNodeContext_ResumedInput(t *testing.T) {
	parent := newMockCtx(t)

	t.Run("nil resumeInputs returns (nil, false)", func(t *testing.T) {
		c := newNodeContext(parent, nil)
		v, ok := c.ResumedInput("any_id")
		if v != nil || ok {
			t.Errorf("ResumedInput() = (%v, %v), want (nil, false)", v, ok)
		}
	})

	t.Run("populated resumeInputs returns matched payload", func(t *testing.T) {
		c := newNodeContext(parent, map[string]any{
			"approval": "yes",
			"comment":  "looks good",
		})
		if v, ok := c.ResumedInput("approval"); !ok || v != "yes" {
			t.Errorf("ResumedInput(\"approval\") = (%v, %v), want (\"yes\", true)", v, ok)
		}
	})

	t.Run("unmatched InterruptID returns (nil, false)", func(t *testing.T) {
		c := newNodeContext(parent, map[string]any{"approval": "yes"})
		if v, ok := c.ResumedInput("missing"); v != nil || ok {
			t.Errorf("ResumedInput(\"missing\") = (%v, %v), want (nil, false)", v, ok)
		}
	})
}

func TestNodeContext_PathAndRunID(t *testing.T) {
	t.Run("top-level static returns empty", func(t *testing.T) {
		c := newNodeContext(newMockCtx(t), nil)
		if got := c.Path(); got != "" {
			t.Errorf("Path() = %q, want empty", got)
		}
		if got := c.RunID(); got != "" {
			t.Errorf("RunID() = %q, want empty", got)
		}
	})

	t.Run("child populated from constructor", func(t *testing.T) {
		parent := newNodeContext(newMockCtx(t), nil)
		child := newChildNodeContext(parent, "wf/fixer@2", "2", nil)
		if got, want := child.Path(), "wf/fixer@2"; got != want {
			t.Errorf("Path() = %q, want %q", got, want)
		}
		if got, want := child.RunID(), "2"; got != want {
			t.Errorf("RunID() = %q, want %q", got, want)
		}
	})
}

func TestNodeContext_ChildInheritsResumeInputs(t *testing.T) {
	parent := newNodeContext(newMockCtx(t), map[string]any{"approval": "yes"})
	child := newChildNodeContext(parent, "wf/asker@1", "1", nil)
	if v, ok := child.ResumedInput("approval"); !ok || v != "yes" {
		t.Errorf("child.ResumedInput(\"approval\") = (%v, %v), want (\"yes\", true)", v, ok)
	}
}

func TestNodeContext_DynamicActivation(t *testing.T) {
	parent := newNodeContext(newMockCtx(t), map[string]any{"approval": "yes"})
	sub := &dynamicSubScheduler{}

	act := newDynamicActivationContext(parent, "city_workflow", sub)

	if got, want := act.Path(), "city_workflow"; got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
	if act.subScheduler != sub {
		t.Errorf("subScheduler pointer differs from supplied")
	}
	if v, ok := act.ResumedInput("approval"); !ok || v != "yes" {
		t.Errorf("ResumedInput inheritance broken: got (%v, %v)", v, ok)
	}
}

func TestNodeContext_WithGoCtx(t *testing.T) {
	parent := newMockCtx(t)
	c := newNodeContext(parent, map[string]any{"approval": "yes"})

	swappedCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	swapped := c.WithGoCtx(swappedCtx)

	t.Run("returns a new instance", func(t *testing.T) {
		if any(swapped) == any(c) {
			t.Errorf("WithGoCtx returned the same NodeContext")
		}
	})

	t.Run("underlying ctx is swapped", func(t *testing.T) {
		// InvocationContext embeds context.Context; verify swap by
		// cancelling swappedCtx and observing Done propagation.
		cancel()
		select {
		case <-swapped.Done():
		default:
			t.Errorf("swapped NodeContext did not observe cancellation")
		}
		select {
		case <-c.Done():
			t.Errorf("original NodeContext was cancelled; WithGoCtx must not mutate")
		default:
		}
	})

	t.Run("resume inputs preserved", func(t *testing.T) {
		if v, ok := swapped.ResumedInput("approval"); !ok || v != "yes" {
			t.Errorf("ResumedInput after WithGoCtx = (%v, %v), want (\"yes\", true)", v, ok)
		}
	})

	t.Run("invocation metadata preserved", func(t *testing.T) {
		if got := swapped.InvocationID(); got != parent.InvocationID() {
			t.Errorf("InvocationID after WithGoCtx = %q, want %q", got, parent.InvocationID())
		}
	})
}

func TestNodeContext_WithGoCtx_PreservesPathAndRunID(t *testing.T) {
	parent := newNodeContext(newMockCtx(t), nil)
	child := newChildNodeContext(parent, "wf/fixer@2", "2", nil)

	swappedCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	swapped := child.WithGoCtx(swappedCtx)

	if got, want := swapped.Path(), "wf/fixer@2"; got != want {
		t.Errorf("Path after WithGoCtx = %q, want %q", got, want)
	}
	if got, want := swapped.RunID(), "2"; got != want {
		t.Errorf("RunID after WithGoCtx = %q, want %q", got, want)
	}
}
