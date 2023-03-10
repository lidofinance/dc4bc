package wc_rotation

import (
	"encoding/hex"
	"testing"
)

func TestGetSigningRoot(t *testing.T) {
	type args struct {
		validatorIndex uint64
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				validatorIndex: 393395,
			},
			// Test for production. Don't merge this branch into master
			want:    `23ccffc7767e1b9a54b3e18c986f00d0345825bcab21eae5fe92c849d6cfedb4`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSigningRoot(tt.args.validatorIndex)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSigningRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			actual := hex.EncodeToString(got[:])

			if actual != tt.want {
				t.Errorf("GetSigningRoot() got = %v, want %v", actual, tt.want)
			}
		})
	}
}
