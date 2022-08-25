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
				223, 44, 104, 73, 126, 224, 239, 4, 247, 97, 211, 159, 123, 90, 221, 190, 243, 40, 38, 154, 218, 168, 142, 241, 81, 126, 54, 48, 139, 156, 197, 108,
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
