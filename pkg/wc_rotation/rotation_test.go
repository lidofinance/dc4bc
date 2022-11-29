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
			want:    `22ff29dacce459cab9b81242127c9dbb54d2d2522b88e88dd2957fec635736a6`,
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
				t.Errorf("GetSigningRoot() got = %v, want %v", got, tt.want)
			}
		})
	}
}
