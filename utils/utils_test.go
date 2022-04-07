package utils

import "testing"

func TestGetConnectionOfJobAndJobTemplate(t *testing.T) {
	type args struct {
		namespace string
		name      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestGetConnectionOfJobAndJobTemplate",
			args: args{
				namespace: "default",
				name:      "flow",
			},
			want: "default.flow",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetConnectionOfJobAndJobTemplate(tt.args.namespace, tt.args.name); got != tt.want {
				t.Errorf("GetConnectionOfJobAndJobTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}
