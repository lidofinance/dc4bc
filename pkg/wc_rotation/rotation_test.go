package wc_rotation

import (
	"encoding/hex"
	"testing"
)

// The same result we got on TS variant
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
			want:    `2202563bc9261c6371fa133be05da0bd42762a7df037a1da7d69d4f756528eb0`,
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
