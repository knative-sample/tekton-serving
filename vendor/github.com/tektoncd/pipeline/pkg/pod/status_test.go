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

package pod

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/logging"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

var ignoreVolatileTime = cmp.Comparer(func(_, _ apis.VolatileTime) bool { return true })

func TestMakeTaskRunStatus(t *testing.T) {
	conditionRunning := apis.Condition{
		Type:    apis.ConditionSucceeded,
		Status:  corev1.ConditionUnknown,
		Reason:  v1beta1.TaskRunReasonRunning.String(),
		Message: "Not all Steps in the Task have finished executing",
	}
	for _, c := range []struct {
		desc      string
		podStatus corev1.PodStatus
		taskSpec  v1beta1.TaskSpec
		want      v1beta1.TaskRunStatus
	}{{
		desc:      "empty",
		podStatus: corev1.PodStatus{},

		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps:    []v1beta1.StepState{},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "ignore-creds-init",
		podStatus: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{{
				// creds-init; ignored
				ImageID: "ignored",
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:    "step-state-name",
				ImageID: "",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					},
				},
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 123,
						}},
					Name:          "state-name",
					ContainerName: "step-state-name",
				}},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "ignore-init-containers",
		podStatus: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{{
				// creds-init; ignored.
				ImageID: "ignoreme",
			}, {
				// git-init; ignored.
				ImageID: "ignoreme",
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:    "step-state-name",
				ImageID: "image-id",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					},
				},
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 123,
						}},
					Name:          "state-name",
					ContainerName: "step-state-name",
					ImageID:       "image-id",
				}},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "success",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-step-push",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 0,
					},
				},
				ImageID: "image-id",
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionTrue,
					Reason:  v1beta1.TaskRunReasonSuccessful.String(),
					Message: "All Steps have completed executing",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 0,
						}},
					Name:          "step-push",
					ContainerName: "step-step-push",
					ImageID:       "image-id",
				}},
				Sidecars: []v1beta1.SidecarState{},
				// We don't actually care about the time, just that it's not nil
				CompletionTime: &metav1.Time{Time: time.Now()},
			},
		},
	}, {
		desc: "running",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-running-step",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
					Name:          "running-step",
					ContainerName: "step-running-step",
				}},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "failure-terminated",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodFailed,
			InitContainerStatuses: []corev1.ContainerStatus{{
				// creds-init status; ignored
				ImageID: "ignore-me",
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:    "step-failure",
				ImageID: "image-id",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 123,
					},
				},
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.TaskRunReasonFailed.String(),
					Message: "\"step-failure\" exited with code 123 (image: \"image-id\"); for logs run: kubectl -n foo logs pod -c step-failure\n",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 123,
						}},

					Name:          "failure",
					ContainerName: "step-failure",
					ImageID:       "image-id",
				}},
				Sidecars: []v1beta1.SidecarState{},
				// We don't actually care about the time, just that it's not nil
				CompletionTime: &metav1.Time{Time: time.Now()},
			},
		},
	}, {
		desc: "failure-message",
		podStatus: corev1.PodStatus{
			Phase:   corev1.PodFailed,
			Message: "boom",
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.TaskRunReasonFailed.String(),
					Message: "boom",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps:    []v1beta1.StepState{},
				Sidecars: []v1beta1.SidecarState{},
				// We don't actually care about the time, just that it's not nil
				CompletionTime: &metav1.Time{Time: time.Now()},
			},
		},
	}, {
		desc: "failed with OOM",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodSucceeded,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-step-push",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						Reason:   "OOMKilled",
						ExitCode: 0,
					},
				},
				ImageID: "image-id",
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.TaskRunReasonFailed.String(),
					Message: "OOMKilled",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:   "OOMKilled",
							ExitCode: 0,
						}},
					Name:          "step-push",
					ContainerName: "step-step-push",
					ImageID:       "image-id",
				}},
				Sidecars: []v1beta1.SidecarState{},
				// We don't actually care about the time, just that it's not nil
				CompletionTime: &metav1.Time{Time: time.Now()},
			},
		},
	}, {
		desc:      "failure-unspecified",
		podStatus: corev1.PodStatus{Phase: corev1.PodFailed},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.TaskRunReasonFailed.String(),
					Message: "build failed for unspecified reasons.",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps:    []v1beta1.StepState{},
				Sidecars: []v1beta1.SidecarState{},
				// We don't actually care about the time, just that it's not nil
				CompletionTime: &metav1.Time{Time: time.Now()},
			},
		},
	}, {
		desc: "pending-waiting-message",
		podStatus: corev1.PodStatus{
			Phase:                 corev1.PodPending,
			InitContainerStatuses: []corev1.ContainerStatus{{
				// creds-init status; ignored
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-status-name",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Message: "i'm pending",
					},
				},
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: `build step "step-status-name" is pending with reason "i'm pending"`,
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Message: "i'm pending",
						},
					},
					Name:          "status-name",
					ContainerName: "step-status-name",
				}},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "pending-pod-condition",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Status:  corev1.ConditionUnknown,
				Type:    "the type",
				Message: "the message",
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: `pod status "the type":"Unknown"; message: "the message"`,
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps:    []v1beta1.StepState{},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "pending-message",
		podStatus: corev1.PodStatus{
			Phase:   corev1.PodPending,
			Message: "pod status message",
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: "pod status message",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps:    []v1beta1.StepState{},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc:      "pending-no-message",
		podStatus: corev1.PodStatus{Phase: corev1.PodPending},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  "Pending",
					Message: "Pending",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps:    []v1beta1.StepState{},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "pending-not-enough-node-resources",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Reason:  corev1.PodReasonUnschedulable,
				Message: "0/1 nodes are available: 1 Insufficient cpu.",
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  ReasonExceededNodeResources,
					Message: "TaskRun Pod exceeded available resources",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps:    []v1beta1.StepState{},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "pending-CreateContainerConfigError",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{{
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason: "CreateContainerConfigError",
					},
				},
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionUnknown,
					Reason:  ReasonCreateContainerConfigError,
					Message: "Pending",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps:    []v1beta1.StepState{},
				Sidecars: []v1beta1.SidecarState{},
			},
		},
	}, {
		desc: "with-sidecar-running",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-running-step",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}, {
				Name:    "sidecar-running",
				ImageID: "image-id",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
				Ready: true,
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
					Name:          "running-step",
					ContainerName: "step-running-step",
				}},
				Sidecars: []v1beta1.SidecarState{{
					ContainerState: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
					Name:          "running",
					ImageID:       "image-id",
					ContainerName: "sidecar-running",
				}},
			},
		},
	}, {
		desc: "with-sidecar-waiting",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-waiting-step",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason:  "PodInitializing",
						Message: "PodInitializing",
					},
				},
			}, {
				Name:    "sidecar-waiting",
				ImageID: "image-id",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason:  "PodInitializing",
						Message: "PodInitializing",
					},
				},
				Ready: true,
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "PodInitializing",
							Message: "PodInitializing",
						},
					},
					Name:          "waiting-step",
					ContainerName: "step-waiting-step",
				}},
				Sidecars: []v1beta1.SidecarState{{
					ContainerState: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "PodInitializing",
							Message: "PodInitializing",
						},
					},
					Name:          "waiting",
					ImageID:       "image-id",
					ContainerName: "sidecar-waiting",
				}},
			},
		},
	}, {
		desc: "with-sidecar-terminated",
		podStatus: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "step-running-step",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}, {
				Name:    "sidecar-error",
				ImageID: "image-id",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 1,
						Reason:   "Error",
						Message:  "Error",
					},
				},
				Ready: true,
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{conditionRunning},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
					Name:          "running-step",
					ContainerName: "step-running-step",
				}},
				Sidecars: []v1beta1.SidecarState{{
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 1,
							Reason:   "Error",
							Message:  "Error",
						},
					},
					Name:          "error",
					ImageID:       "image-id",
					ContainerName: "sidecar-error",
				}},
			},
		},
	}, {
		desc: "non-json-termination-message-with-steps-afterwards-shouldnt-panic",
		taskSpec: v1beta1.TaskSpec{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name: "non-json",
			}}, {Container: corev1.Container{
				Name: "after-non-json",
			}}, {Container: corev1.Container{
				Name: "this-step-might-panic",
			}}, {Container: corev1.Container{
				Name: "foo",
			}}},
		},
		podStatus: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:    "step-this-step-might-panic",
				ImageID: "image",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{},
				},
			}, {
				Name:    "step-foo",
				ImageID: "image",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{},
				},
			}, {
				Name:    "step-non-json",
				ImageID: "image",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 1,
						Message:  "this is a non-json termination message. dont panic!",
					},
				},
			}, {
				Name:    "step-after-non-json",
				ImageID: "image",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{},
				},
			}},
		},
		want: v1beta1.TaskRunStatus{
			Status: duckv1beta1.Status{
				Conditions: []apis.Condition{{
					Type:    apis.ConditionSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  v1beta1.TaskRunReasonFailed.String(),
					Message: "\"step-non-json\" exited with code 1 (image: \"image\"); for logs run: kubectl -n foo logs pod -c step-non-json\n",
				}},
			},
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Steps: []v1beta1.StepState{{
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 1,
							Message:  "this is a non-json termination message. dont panic!",
						}},

					Name:          "non-json",
					ContainerName: "step-non-json",
					ImageID:       "image",
				}, {
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{}},
					Name:          "after-non-json",
					ContainerName: "step-after-non-json",
					ImageID:       "image",
				}, {
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{}},
					Name:          "this-step-might-panic",
					ContainerName: "step-this-step-might-panic",
					ImageID:       "image",
				}, {
					ContainerState: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{}},
					Name:          "foo",
					ContainerName: "step-foo",
					ImageID:       "image",
				}},
				Sidecars: []v1beta1.SidecarState{},
				// We don't actually care about the time, just that it's not nil
				CompletionTime: &metav1.Time{Time: time.Now()},
			},
		},
	}} {
		t.Run(c.desc, func(t *testing.T) {
			now := metav1.Now()
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod",
					Namespace:         "foo",
					CreationTimestamp: now,
				},
				Status: c.podStatus,
			}
			startTime := time.Date(2010, 1, 1, 1, 1, 1, 1, time.UTC)
			tr := v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "task-run",
					Namespace: "foo",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						StartTime: &metav1.Time{Time: startTime},
					},
				},
			}

			logger, _ := logging.NewLogger("", "status")
			got := MakeTaskRunStatus(logger, tr, pod, c.taskSpec)

			// Common traits, set for test case brevity.
			c.want.PodName = "pod"
			c.want.StartTime = &metav1.Time{Time: startTime}

			ensureTimeNotNil := cmp.Comparer(func(x, y *metav1.Time) bool {
				if x == nil {
					return y == nil
				}
				return y != nil
			})
			if d := cmp.Diff(c.want, got, ignoreVolatileTime, ensureTimeNotNil); d != "" {
				t.Errorf("Diff %s", diff.PrintWantGot(d))
			}
			if tr.Status.StartTime.Time != c.want.StartTime.Time {
				t.Errorf("Expected TaskRun startTime to be unchanged but was %s", tr.Status.StartTime)
			}
		})
	}
}

func TestSidecarsReady(t *testing.T) {
	for _, c := range []struct {
		desc     string
		statuses []corev1.ContainerStatus
		want     bool
	}{{
		desc: "no sidecars",
		statuses: []corev1.ContainerStatus{
			{Name: "step-ignore-me"},
			{Name: "step-ignore-me"},
			{Name: "step-ignore-me"},
		},
		want: true,
	}, {
		desc: "both sidecars ready",
		statuses: []corev1.ContainerStatus{
			{Name: "step-ignore-me"},
			{
				Name:  "sidecar-bar",
				Ready: true,
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{
						StartedAt: metav1.NewTime(time.Now()),
					},
				},
			},
			{Name: "step-ignore-me"},
			{
				Name: "sidecar-stopped-baz",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 99,
					},
				},
			},
			{Name: "step-ignore-me"},
		},
		want: true,
	}, {
		desc: "one sidecar ready, one not running",
		statuses: []corev1.ContainerStatus{
			{Name: "step-ignore-me"},
			{
				Name:  "sidecar-ready",
				Ready: true,
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{
						StartedAt: metav1.NewTime(time.Now()),
					},
				},
			},
			{Name: "step-ignore-me"},
			{
				Name: "sidecar-unready",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{},
				},
			},
			{Name: "step-ignore-me"},
		},
		want: false,
	}, {
		desc: "one sidecar running but not ready",
		statuses: []corev1.ContainerStatus{
			{Name: "step-ignore-me"},
			{
				Name:  "sidecar-running-not-ready",
				Ready: false, // Not ready.
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{
						StartedAt: metav1.NewTime(time.Now()),
					},
				},
			},
			{Name: "step-ignore-me"},
		},
		want: false,
	}} {
		t.Run(c.desc, func(t *testing.T) {
			got := SidecarsReady(corev1.PodStatus{
				Phase:             corev1.PodRunning,
				ContainerStatuses: c.statuses,
			})
			if got != c.want {
				t.Errorf("SidecarsReady got %t, want %t", got, c.want)
			}
		})
	}
}

func TestSortTaskRunStepOrder(t *testing.T) {
	steps := []v1beta1.Step{{Container: corev1.Container{
		Name: "hello",
	}}, {Container: corev1.Container{
		Name: "exit",
	}}, {Container: corev1.Container{
		Name: "world",
	}}}

	stepStates := []v1beta1.StepState{{
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 0,
				Reason:   "Completed",
			},
		},
		Name: "world",
	}, {
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 1,
				Reason:   "Error",
			},
		},
		Name: "exit",
	}, {
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 0,
				Reason:   "Completed",
			},
		},
		Name: "hello",
	}, {
		ContainerState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 0,
				Reason:   "Completed",
			},
		},
		Name: "nop",
	}}

	gotStates := sortTaskRunStepOrder(stepStates, steps)
	var gotNames []string
	for _, g := range gotStates {
		gotNames = append(gotNames, g.Name)
	}

	want := []string{"hello", "exit", "world", "nop"}
	if d := cmp.Diff(want, gotNames); d != "" {
		t.Errorf("Unexpected step order %s", diff.PrintWantGot(d))
	}
}

func TestSortContainerStatuses(t *testing.T) {
	samplePod := corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "hello",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							FinishedAt: metav1.Time{Time: time.Now()},
						},
					},
				}, {
					Name:  "my",
					State: corev1.ContainerState{
						// No Terminated status, terminated == 0 (and no panic)
					},
				}, {
					Name: "world",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							FinishedAt: metav1.Time{Time: time.Now().Add(time.Second * -5)},
						},
					},
				},
			},
		},
	}
	SortContainerStatuses(&samplePod)
	var gotNames []string
	for _, status := range samplePod.Status.ContainerStatuses {
		gotNames = append(gotNames, status.Name)
	}

	want := []string{"my", "world", "hello"}
	if d := cmp.Diff(want, gotNames); d != "" {
		t.Errorf("Unexpected step order %s", diff.PrintWantGot(d))
	}

}
