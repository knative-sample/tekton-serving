/*
Copyright 2020 The Tetkon Authors

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

package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	tb "github.com/tektoncd/pipeline/internal/builder/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/test/diff"
)

func TestConditionSpec_SetDefaults(t *testing.T) {
	cases := []struct {
		name   string
		input  *v1alpha1.ConditionSpec
		output *v1alpha1.ConditionSpec
	}{
		{
			name:   "No Param",
			input:  &v1alpha1.ConditionSpec{},
			output: &v1alpha1.ConditionSpec{},
		},
		{
			name: "One Param",
			input: &v1alpha1.ConditionSpec{
				Params: []v1alpha1.ParamSpec{
					{
						Name:    "test-1",
						Default: tb.ArrayOrString("an", "array"),
					},
				},
			},
			output: &v1alpha1.ConditionSpec{
				Params: []v1alpha1.ParamSpec{
					{
						Name:    "test-1",
						Type:    v1alpha1.ParamTypeArray,
						Default: tb.ArrayOrString("an", "array"),
					},
				},
			},
		},
		{
			name: "Multiple Param",
			input: &v1alpha1.ConditionSpec{
				Params: []v1alpha1.ParamSpec{
					{
						Name:    "test-1",
						Default: tb.ArrayOrString("array"),
					},
					{
						Name:    "test-2",
						Default: tb.ArrayOrString("an", "array"),
					},
				},
			},
			output: &v1alpha1.ConditionSpec{
				Params: []v1alpha1.ParamSpec{
					{
						Name:    "test-1",
						Type:    v1alpha1.ParamTypeString,
						Default: tb.ArrayOrString("array"),
					},
					{
						Name:    "test-2",
						Type:    v1alpha1.ParamTypeArray,
						Default: tb.ArrayOrString("an", "array"),
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.input.SetDefaults(ctx)
			if d := cmp.Diff(tc.output, tc.input); d != "" {
				t.Errorf("Mismatch of PipelineRunSpec: %s", diff.PrintWantGot(d))
			}
		})
	}
}
