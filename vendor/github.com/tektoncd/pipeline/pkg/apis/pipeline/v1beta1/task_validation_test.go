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

package v1beta1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	tb "github.com/tektoncd/pipeline/internal/builder/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

var validResource = v1beta1.TaskResource{
	ResourceDeclaration: v1beta1.ResourceDeclaration{
		Name: "source",
		Type: "git",
	},
}

var invalidResource = v1beta1.TaskResource{
	ResourceDeclaration: v1beta1.ResourceDeclaration{
		Name: "source",
		Type: "what",
	},
}

var validSteps = []v1beta1.Step{{Container: corev1.Container{
	Name:  "mystep",
	Image: "myimage",
}}}

var invalidSteps = []v1beta1.Step{{Container: corev1.Container{
	Name:  "replaceImage",
	Image: "myimage",
}}}

func TestTaskSpecValidate(t *testing.T) {
	type fields struct {
		Params       []v1beta1.ParamSpec
		Resources    *v1beta1.TaskResources
		Steps        []v1beta1.Step
		StepTemplate *corev1.Container
		Workspaces   []v1beta1.WorkspaceDeclaration
		Results      []v1beta1.TaskResult
	}
	tests := []struct {
		name   string
		fields fields
	}{{
		name: "unnamed steps",
		fields: fields{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Image: "myimage",
			}}, {Container: corev1.Container{
				Image: "myotherimage",
			}}},
		},
	}, {
		name: "valid input resources",
		fields: fields{
			Resources: &v1beta1.TaskResources{
				Inputs: []v1beta1.TaskResource{validResource},
			},
			Steps: validSteps,
		},
	}, {
		name: "valid output resources",
		fields: fields{
			Resources: &v1beta1.TaskResources{
				Outputs: []v1beta1.TaskResource{validResource},
			},
			Steps: validSteps,
		},
	}, {
		name: "valid params type implied",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name:        "task",
				Description: "param",
				Default:     tb.ArrayOrString("default"),
			}},
			Steps: validSteps,
		},
	}, {
		name: "valid params type explicit",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name:        "task",
				Type:        v1beta1.ParamTypeString,
				Description: "param",
				Default:     tb.ArrayOrString("default"),
			}},
			Steps: validSteps,
		},
	}, {
		name: "valid template variable",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
			}, {
				Name: "foo-is-baz",
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "url",
				Args:       []string{"--flag=$(params.baz) && $(params.foo-is-baz)"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
	}, {
		name: "valid array template variable",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Type: v1beta1.ParamTypeArray,
			}, {
				Name: "foo-is-baz",
				Type: v1beta1.ParamTypeArray,
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "myimage",
				Command:    []string{"$(params.foo-is-baz)"},
				Args:       []string{"$(params.baz)", "middle string", "$(params.foo-is-baz)"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
	}, {
		name: "valid star array template variable",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Type: v1beta1.ParamTypeArray,
			}, {
				Name: "foo-is-baz",
				Type: v1beta1.ParamTypeArray,
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "myimage",
				Command:    []string{"$(params.foo-is-baz)"},
				Args:       []string{"$(params.baz[*])", "middle string", "$(params.foo-is-baz[*])"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
	}, {
		name: "valid creds-init path variable",
		fields: fields{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:  "mystep",
				Image: "echo",
				Args:  []string{"$(credentials.path)"},
			}}},
		},
	}, {
		name: "step template included in validation",
		fields: fields{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:    "astep",
				Command: []string{"echo"},
				Args:    []string{"hello"},
			}}},
			StepTemplate: &corev1.Container{
				Image: "some-image",
			},
		},
	}, {
		name: "valid step with script",
		fields: fields{
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image: "my-image",
				},
				Script: `
				#!/usr/bin/env bash
				hello world`,
			}},
		},
	}, {
		name: "valid step with parameterized script",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
			}, {
				Name: "foo-is-baz",
			}},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image: "my-image",
				},
				Script: `
					#!/usr/bin/env bash
					hello $(params.baz)`,
			}},
		},
	}, {
		name: "valid step with script and args",
		fields: fields{
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image: "my-image",
					Args:  []string{"arg"},
				},
				Script: `
				#!/usr/bin/env  bash
				hello $1`,
			}},
		},
	}, {
		name: "valid step with volumeMount under /tekton/home",
		fields: fields{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Image: "myimage",
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "foo",
					MountPath: "/tekton/home",
				}},
			}}},
		},
	}, {
		name: "valid workspace",
		fields: fields{
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image: "my-image",
					Args:  []string{"arg"},
				},
			}},
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name:        "foo-workspace",
				Description: "my great workspace",
				MountPath:   "some/path",
			}},
		},
	}, {
		name: "valid result",
		fields: fields{
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image: "my-image",
					Args:  []string{"arg"},
				},
			}},
			Results: []v1beta1.TaskResult{{
				Name:        "MY-RESULT",
				Description: "my great result",
			}},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &v1beta1.TaskSpec{
				Params:       tt.fields.Params,
				Resources:    tt.fields.Resources,
				Steps:        tt.fields.Steps,
				StepTemplate: tt.fields.StepTemplate,
				Workspaces:   tt.fields.Workspaces,
				Results:      tt.fields.Results,
			}
			ctx := context.Background()
			ts.SetDefaults(ctx)
			if err := ts.Validate(ctx); err != nil {
				t.Errorf("TaskSpec.Validate() = %v", err)
			}
		})
	}
}

func TestTaskSpecValidateError(t *testing.T) {
	type fields struct {
		Params       []v1beta1.ParamSpec
		Resources    *v1beta1.TaskResources
		Steps        []v1beta1.Step
		Volumes      []corev1.Volume
		StepTemplate *corev1.Container
		Workspaces   []v1beta1.WorkspaceDeclaration
		Results      []v1beta1.TaskResult
	}
	tests := []struct {
		name          string
		fields        fields
		expectedError apis.FieldError
	}{{
		name: "empty spec",
		expectedError: apis.FieldError{
			Message: `missing field(s)`,
			Paths:   []string{"steps"},
		},
	}, {
		name: "no step",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name:        "validparam",
				Type:        v1beta1.ParamTypeString,
				Description: "parameter",
				Default:     tb.ArrayOrString("default"),
			}},
		},
		expectedError: apis.FieldError{
			Message: `missing field(s)`,
			Paths:   []string{"steps"},
		},
	}, {
		name: "invalid input resource",
		fields: fields{
			Resources: &v1beta1.TaskResources{
				Inputs: []v1beta1.TaskResource{invalidResource},
			},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `invalid value: what`,
			Paths:   []string{"taskspec.resources.inputs.source.type"},
		},
	}, {
		name: "one invalid input resource",
		fields: fields{
			Resources: &v1beta1.TaskResources{
				Inputs: []v1beta1.TaskResource{validResource, invalidResource},
			},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `invalid value: what`,
			Paths:   []string{"taskspec.resources.inputs.source.type"},
		},
	}, {
		name: "duplicated inputs resources",
		fields: fields{
			Resources: &v1beta1.TaskResources{
				Inputs:  []v1beta1.TaskResource{validResource, validResource},
				Outputs: []v1beta1.TaskResource{validResource},
			},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `expected exactly one, got both`,
			Paths:   []string{"taskspec.resources.inputs.name"},
		},
	}, {
		name: "invalid output resource",
		fields: fields{
			Resources: &v1beta1.TaskResources{
				Outputs: []v1beta1.TaskResource{invalidResource},
			},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `invalid value: what`,
			Paths:   []string{"taskspec.resources.outputs.source.type"},
		},
	}, {
		name: "one invalid output resource",
		fields: fields{
			Resources: &v1beta1.TaskResources{
				Outputs: []v1beta1.TaskResource{validResource, invalidResource},
			},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `invalid value: what`,
			Paths:   []string{"taskspec.resources.outputs.source.type"},
		},
	}, {
		name: "duplicated outputs resources",
		fields: fields{
			Resources: &v1beta1.TaskResources{
				Inputs:  []v1beta1.TaskResource{validResource},
				Outputs: []v1beta1.TaskResource{validResource, validResource},
			},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `expected exactly one, got both`,
			Paths:   []string{"taskspec.resources.outputs.name"},
		},
	}, {
		name: "invalid param type",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name:        "validparam",
				Type:        v1beta1.ParamTypeString,
				Description: "parameter",
				Default:     tb.ArrayOrString("default"),
			}, {
				Name:        "param-with-invalid-type",
				Type:        "invalidtype",
				Description: "invalidtypedesc",
				Default:     tb.ArrayOrString("default"),
			}},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `invalid value: invalidtype`,
			Paths:   []string{"taskspec.params.param-with-invalid-type.type"},
		},
	}, {
		name: "param mismatching default/type 1",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name:        "task",
				Type:        v1beta1.ParamTypeArray,
				Description: "param",
				Default:     tb.ArrayOrString("default"),
			}},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `"array" type does not match default value's type: "string"`,
			Paths:   []string{"taskspec.params.task.type", "taskspec.params.task.default.type"},
		},
	}, {
		name: "param mismatching default/type 2",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name:        "task",
				Type:        v1beta1.ParamTypeString,
				Description: "param",
				Default:     tb.ArrayOrString("default", "array"),
			}},
			Steps: validSteps,
		},
		expectedError: apis.FieldError{
			Message: `"string" type does not match default value's type: "array"`,
			Paths:   []string{"taskspec.params.task.type", "taskspec.params.task.default.type"},
		},
	}, {
		name: "invalid step",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name:        "validparam",
				Type:        v1beta1.ParamTypeString,
				Description: "parameter",
				Default:     tb.ArrayOrString("default"),
			}},
			Steps: []v1beta1.Step{},
		},
		expectedError: apis.FieldError{
			Message: "missing field(s)",
			Paths:   []string{"steps"},
		},
	}, {
		name: "invalid step name",
		fields: fields{
			Steps: invalidSteps,
		},
		expectedError: apis.FieldError{
			Message: `invalid value "replaceImage"`,
			Paths:   []string{"taskspec.steps.name"},
			Details: "Task step name must be a valid DNS Label, For more info refer to https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
		},
	}, {
		name: "inexistent param variable",
		fields: fields{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:  "mystep",
				Image: "myimage",
				Args:  []string{"--flag=$(params.inexistent)"},
			}}},
		},
		expectedError: apis.FieldError{
			Message: `non-existent variable in "--flag=$(params.inexistent)" for step arg[0]`,
			Paths:   []string{"taskspec.steps.arg[0]"},
		},
	}, {
		name: "array used in unaccepted field",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Type: v1beta1.ParamTypeArray,
			}, {
				Name: "foo-is-baz",
				Type: v1beta1.ParamTypeArray,
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "$(params.baz)",
				Command:    []string{"$(params.foo-is-baz)"},
				Args:       []string{"$(params.baz)", "middle string", "url"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
		expectedError: apis.FieldError{
			Message: `variable type invalid in "$(params.baz)" for step image`,
			Paths:   []string{"taskspec.steps.image"},
		},
	}, {
		name: "array star used in unaccepted field",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Type: v1beta1.ParamTypeArray,
			}, {
				Name: "foo-is-baz",
				Type: v1beta1.ParamTypeArray,
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "$(params.baz[*])",
				Command:    []string{"$(params.foo-is-baz)"},
				Args:       []string{"$(params.baz)", "middle string", "url"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
		expectedError: apis.FieldError{
			Message: `variable type invalid in "$(params.baz[*])" for step image`,
			Paths:   []string{"taskspec.steps.image"},
		},
	}, {
		name: "array star used illegaly in script field",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Type: v1beta1.ParamTypeArray,
			}, {
				Name: "foo-is-baz",
				Type: v1beta1.ParamTypeArray,
			}},
			Steps: []v1beta1.Step{
				{
					Script: "$(params.baz[*])",
					Container: corev1.Container{
						Name:       "mystep",
						Image:      "my-image",
						WorkingDir: "/foo/bar/src/",
					},
				}},
		},
		expectedError: apis.FieldError{
			Message: `variable type invalid in "$(params.baz[*])" for step script`,
			Paths:   []string{"taskspec.steps.script"},
		},
	}, {
		name: "array not properly isolated",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Type: v1beta1.ParamTypeArray,
			}, {
				Name: "foo-is-baz",
				Type: v1beta1.ParamTypeArray,
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "someimage",
				Command:    []string{"$(params.foo-is-baz)"},
				Args:       []string{"not isolated: $(params.baz)", "middle string", "url"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
		expectedError: apis.FieldError{
			Message: `variable is not properly isolated in "not isolated: $(params.baz)" for step arg[0]`,
			Paths:   []string{"taskspec.steps.arg[0]"},
		},
	}, {
		name: "array star not properly isolated",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Type: v1beta1.ParamTypeArray,
			}, {
				Name: "foo-is-baz",
				Type: v1beta1.ParamTypeArray,
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "someimage",
				Command:    []string{"$(params.foo-is-baz)"},
				Args:       []string{"not isolated: $(params.baz[*])", "middle string", "url"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
		expectedError: apis.FieldError{
			Message: `variable is not properly isolated in "not isolated: $(params.baz[*])" for step arg[0]`,
			Paths:   []string{"taskspec.steps.arg[0]"},
		},
	}, {
		name: "inferred array not properly isolated",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Default: &v1beta1.ArrayOrString{
					Type:     v1beta1.ParamTypeArray,
					ArrayVal: []string{"implied", "array", "type"},
				},
			}, {
				Name: "foo-is-baz",
				Default: &v1beta1.ArrayOrString{
					Type:     v1beta1.ParamTypeArray,
					ArrayVal: []string{"implied", "array", "type"},
				},
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "someimage",
				Command:    []string{"$(params.foo-is-baz)"},
				Args:       []string{"not isolated: $(params.baz)", "middle string", "url"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
		expectedError: apis.FieldError{
			Message: `variable is not properly isolated in "not isolated: $(params.baz)" for step arg[0]`,
			Paths:   []string{"taskspec.steps.arg[0]"},
		},
	}, {
		name: "inferred array star not properly isolated",
		fields: fields{
			Params: []v1beta1.ParamSpec{{
				Name: "baz",
				Default: &v1beta1.ArrayOrString{
					Type:     v1beta1.ParamTypeArray,
					ArrayVal: []string{"implied", "array", "type"},
				},
			}, {
				Name: "foo-is-baz",
				Default: &v1beta1.ArrayOrString{
					Type:     v1beta1.ParamTypeArray,
					ArrayVal: []string{"implied", "array", "type"},
				},
			}},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:       "mystep",
				Image:      "someimage",
				Command:    []string{"$(params.foo-is-baz)"},
				Args:       []string{"not isolated: $(params.baz[*])", "middle string", "url"},
				WorkingDir: "/foo/bar/src/",
			}}},
		},
		expectedError: apis.FieldError{
			Message: `variable is not properly isolated in "not isolated: $(params.baz[*])" for step arg[0]`,
			Paths:   []string{"taskspec.steps.arg[0]"},
		},
	}, {
		name: "Inexistent param variable with existing",
		fields: fields{
			Params: []v1beta1.ParamSpec{
				{
					Name:        "foo",
					Description: "param",
					Default:     tb.ArrayOrString("default"),
				},
			},
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Name:  "mystep",
				Image: "myimage",
				Args:  []string{"$(params.foo) && $(params.inexistent)"},
			}}},
		},
		expectedError: apis.FieldError{
			Message: `non-existent variable in "$(params.foo) && $(params.inexistent)" for step arg[0]`,
			Paths:   []string{"taskspec.steps.arg[0]"},
		},
	}, {
		name: "Multiple volumes with same name",
		fields: fields{
			Steps: validSteps,
			Volumes: []corev1.Volume{{
				Name: "workspace",
			}, {
				Name: "workspace",
			}},
		},
		expectedError: apis.FieldError{
			Message: `multiple volumes with same name "workspace"`,
			Paths:   []string{"volumes.name"},
		},
	}, {
		name: "step with script and command",
		fields: fields{
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image:   "myimage",
					Command: []string{"command"},
				},
				Script: "script",
			}},
		},
		expectedError: apis.FieldError{
			Message: "step 0 script cannot be used with command",
			Paths:   []string{"steps.script"},
		},
	}, {
		name: "step volume mounts under /tekton/",
		fields: fields{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Image: "myimage",
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "foo",
					MountPath: "/tekton/foo",
				}},
			}}},
		},
		expectedError: apis.FieldError{
			Message: `step 0 volumeMount cannot be mounted under /tekton/ (volumeMount "foo" mounted at "/tekton/foo")`,
			Paths:   []string{"steps.volumeMounts.mountPath"},
		},
	}, {
		name: "step volume mount name starts with tekton-internal-",
		fields: fields{
			Steps: []v1beta1.Step{{Container: corev1.Container{
				Image: "myimage",
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "tekton-internal-foo",
					MountPath: "/this/is/fine",
				}},
			}}},
		},
		expectedError: apis.FieldError{
			Message: `step 0 volumeMount name "tekton-internal-foo" cannot start with "tekton-internal-"`,
			Paths:   []string{"steps.volumeMounts.name"},
		},
	}, {
		name: "declared workspaces names are not unique",
		fields: fields{
			Steps: validSteps,
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name: "same-workspace",
			}, {
				Name: "same-workspace",
			}},
		},
		expectedError: apis.FieldError{
			Message: "workspace name \"same-workspace\" must be unique",
			Paths:   []string{"workspaces.name"},
		},
	}, {
		name: "declared workspaces clash with each other",
		fields: fields{
			Steps: validSteps,
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name:      "some-workspace",
				MountPath: "/foo",
			}, {
				Name:      "another-workspace",
				MountPath: "/foo",
			}},
		},
		expectedError: apis.FieldError{
			Message: "workspace mount path \"/foo\" must be unique",
			Paths:   []string{"workspaces.mountpath"},
		},
	}, {
		name: "workspace mount path already in volumeMounts",
		fields: fields{
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image:   "myimage",
					Command: []string{"command"},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "my-mount",
						MountPath: "/foo",
					}},
				},
			}},
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name:      "some-workspace",
				MountPath: "/foo",
			}},
		},
		expectedError: apis.FieldError{
			Message: "workspace mount path \"/foo\" must be unique",
			Paths:   []string{"workspaces.mountpath"},
		},
	}, {
		name: "workspace default mount path already in volumeMounts",
		fields: fields{
			Steps: []v1beta1.Step{{
				Container: corev1.Container{
					Image:   "myimage",
					Command: []string{"command"},
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "my-mount",
						MountPath: "/workspace/some-workspace/",
					}},
				},
			}},
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name: "some-workspace",
			}},
		},
		expectedError: apis.FieldError{
			Message: "workspace mount path \"/workspace/some-workspace\" must be unique",
			Paths:   []string{"workspaces.mountpath"},
		},
	}, {
		name: "workspace mount path already in stepTemplate",
		fields: fields{
			StepTemplate: &corev1.Container{
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "my-mount",
					MountPath: "/foo",
				}},
			},
			Steps: validSteps,
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name:      "some-workspace",
				MountPath: "/foo",
			}},
		},
		expectedError: apis.FieldError{
			Message: "workspace mount path \"/foo\" must be unique",
			Paths:   []string{"workspaces.mountpath"},
		},
	}, {
		name: "workspace default mount path already in stepTemplate",
		fields: fields{
			StepTemplate: &corev1.Container{
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "my-mount",
					MountPath: "/workspace/some-workspace",
				}},
			},
			Steps: validSteps,
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name: "some-workspace",
			}},
		},
		expectedError: apis.FieldError{
			Message: "workspace mount path \"/workspace/some-workspace\" must be unique",
			Paths:   []string{"workspaces.mountpath"},
		},
	}, {
		name: "result name not validate",
		fields: fields{
			Steps: validSteps,
			Results: []v1beta1.TaskResult{{
				Name:        "MY^RESULT",
				Description: "my great result",
			}},
		},
		expectedError: apis.FieldError{
			Message: `invalid key name "MY^RESULT"`,
			Paths:   []string{"results[0].name"},
			Details: "Name must consist of alphanumeric characters, '-', '_', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my-name',  or 'my_name', regex used for validation is '^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$')",
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &v1beta1.TaskSpec{
				Params:       tt.fields.Params,
				Resources:    tt.fields.Resources,
				Steps:        tt.fields.Steps,
				Volumes:      tt.fields.Volumes,
				StepTemplate: tt.fields.StepTemplate,
				Workspaces:   tt.fields.Workspaces,
				Results:      tt.fields.Results,
			}
			ctx := context.Background()
			ts.SetDefaults(ctx)
			err := ts.Validate(context.Background())
			if err == nil {
				t.Fatalf("Expected an error, got nothing for %v", ts)
			}
			if d := cmp.Diff(tt.expectedError, *err, cmpopts.IgnoreUnexported(apis.FieldError{})); d != "" {
				t.Errorf("TaskSpec.Validate() errors diff %s", diff.PrintWantGot(d))
			}
		})
	}
}
