package frontproxy

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	operatorv1alpha1 "github.com/kcp-dev/kcp-operator/sdk/apis/operator/v1alpha1"
)

func TestGetArgs(t *testing.T) {
	tests := map[string]struct {
		in  *operatorv1alpha1.FrontProxySpec
		exp []string
	}{
		"only defaults configured": {
			in:  &operatorv1alpha1.FrontProxySpec{Auth: &operatorv1alpha1.AuthSpec{}},
			exp: defaultArgs,
		},
		"drop-groups and pass-on-groups configured": {
			in: &operatorv1alpha1.FrontProxySpec{
				Auth: &operatorv1alpha1.AuthSpec{
					DropGroups:   []string{"some-group", "some-other-group"},
					PassOnGroups: []string{"totally-different-group"},
				},
			},
			exp: append(defaultArgs, []string{
				"--authentication-drop-groups=\"some-group,some-other-group\"",
				"--authentication-pass-on-groups=\"totally-different-group\"",
			}...),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			res := getArgs(tc.in)
			if !cmp.Equal(res, tc.exp) {
				t.Error(cmp.Diff(res, tc.exp))
			}
		})
	}
}
