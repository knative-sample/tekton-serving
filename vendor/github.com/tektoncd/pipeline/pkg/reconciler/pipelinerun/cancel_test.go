/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pipelinerun

import (
	"context"
	"testing"

	tb "github.com/tektoncd/pipeline/internal/builder/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestCancelPipelineRun(t *testing.T) {
	testCases := []struct {
		name        string
		pipelineRun *v1beta1.PipelineRun
		taskRuns    []*v1beta1.TaskRun
	}{{
		name: "no-resolved-taskrun",
		pipelineRun: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pipeline-run-cancelled"},
			Spec: v1beta1.PipelineRunSpec{
				Status: v1beta1.PipelineRunSpecStatusCancelled,
			},
		},
	}, {
		name: "1-taskrun",
		pipelineRun: tb.PipelineRun("test-pipeline-run-cancelled", tb.PipelineRunNamespace("foo"),
			tb.PipelineRunSpec("test-pipeline",
				tb.PipelineRunCancelled,
			),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus("t1", &v1beta1.PipelineRunTaskRunStatus{
					PipelineTaskName: "task-1",
				})),
		),
		taskRuns: []*v1beta1.TaskRun{tb.TaskRun("t1", tb.TaskRunNamespace("foo"))},
	}, {
		name: "multiple-taskruns",
		pipelineRun: tb.PipelineRun("test-pipeline-run-cancelled", tb.PipelineRunNamespace("foo"),
			tb.PipelineRunSpec("test-pipeline",
				tb.PipelineRunCancelled,
			),
			tb.PipelineRunStatus(
				tb.PipelineRunTaskRunsStatus(
					"t1", &v1beta1.PipelineRunTaskRunStatus{PipelineTaskName: "task-1"}),
				tb.PipelineRunTaskRunsStatus(
					"t2", &v1beta1.PipelineRunTaskRunStatus{PipelineTaskName: "task-2"})),
		),
		taskRuns: []*v1beta1.TaskRun{tb.TaskRun("t1", tb.TaskRunNamespace("foo")), tb.TaskRun("t2", tb.TaskRunNamespace("foo"))},
	}}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			d := test.Data{
				PipelineRuns: []*v1beta1.PipelineRun{tc.pipelineRun},
				TaskRuns:     tc.taskRuns,
			}
			ctx, _ := ttesting.SetupFakeContext(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			c, _ := test.SeedTestData(t, ctx, d)
			err := cancelPipelineRun(logtesting.TestLogger(t), tc.pipelineRun, c.Pipeline)
			if err != nil {
				t.Fatal(err)
			}
			// This PipelineRun should still be complete and false, and the status should reflect that
			cond := tc.pipelineRun.Status.GetCondition(apis.ConditionSucceeded)
			if cond.IsTrue() {
				t.Errorf("Expected PipelineRun status to be complete and false, but was %v", cond)
			}
			l, err := c.Pipeline.TektonV1beta1().TaskRuns("foo").List(metav1.ListOptions{})
			if err != nil {
				t.Fatal(err)
			}
			for _, tr := range l.Items {
				if tr.Spec.Status != v1beta1.TaskRunSpecStatusCancelled {
					t.Errorf("expected task %q to be marked as cancelled, was %q", tr.Name, tr.Spec.Status)
				}
			}
		})
	}
}
