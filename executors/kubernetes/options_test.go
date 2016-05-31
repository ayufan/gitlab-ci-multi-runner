package kubernetes

import (
	"testing"

	"gitlab.com/gitlab-org/gitlab-ci-multi-runner/common"
)

type FakeOptions struct {
	privileged bool
}

func (f FakeOptions) Privileged() bool {
	return f.privileged
}

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		BuildVariables common.BuildVariables
		Test           func(o Options) interface{}
		Expected       interface{}
	}{
		{
			BuildVariables: common.BuildVariables{
				{
					Key:   "privileged",
					Value: "true",
				},
			},
			Test: func(o Options) interface{} {
				return o.Privileged()
			},
			Expected: true,
		},
		{
			BuildVariables: common.BuildVariables{
				{
					Key:   "privileged",
					Value: "True",
				},
			},
			Test: func(o Options) interface{} {
				return o.Privileged()
			},
			Expected: true,
		},
		{
			BuildVariables: common.BuildVariables{
				{
					Key:   "privileged",
					Value: "TRUE",
				},
			},
			Test: func(o Options) interface{} {
				return o.Privileged()
			},
			Expected: true,
		},
		{
			BuildVariables: common.BuildVariables{
				{
					Key:   "privileged",
					Value: "false",
				},
			},
			Test: func(o Options) interface{} {
				return o.Privileged()
			},
			Expected: false,
		},
		{
			BuildVariables: common.BuildVariables{
				{
					Key:   "privileged",
					Value: "",
				},
			},
			Test: func(o Options) interface{} {
				return o.Privileged()
			},
			Expected: false,
		},
	}

	for _, test := range tests {
		o := DefaultOptions{
			BuildVariables: test.BuildVariables,
		}

		if res := test.Test(o); res != test.Expected {
			t.Errorf("error testing build options. %v != %v", res, test.Expected)
		}
	}
}
