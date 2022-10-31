package wc_rotation

import (
	"reflect"
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
		want    [32]byte
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				validatorIndex: 52793,
			},
			want: [32]byte{
				177, 122, 243, 237, 155, 72, 202, 9, 3, 132, 122, 193, 247, 215, 19, 0, 4, 76, 147, 121, 183, 215, 188, 24, 249, 103, 175, 3, 160, 41, 239, 35,
			},
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSigningRoot() got = %v, want %v", got, tt.want)
			}
		})
	}
}
