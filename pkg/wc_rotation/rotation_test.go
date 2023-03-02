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
			want:    `5c57b22ed4078f4e4e8ec3188c8c25895154a55c62525488f40b37a3464da6fe`,
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
