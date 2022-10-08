package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AddressType_Unmarshal(t *testing.T) {
	tests := []struct {
		arg     string
		want    Address
		wantErr bool
	}{
		{
			arg:  `"0x00112233445566778899aabbccddeeff00112233"`,
			want: (Address)([AddressLength]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33}),
		},
		{
			arg:  `"00112233445566778899aabbccddeeff00112233"`,
			want: (Address)([AddressLength]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33}),
		},
		{
			arg:     `"00112233445566778899aabbccddeeff0011223344"`,
			wantErr: true,
		},
		{
			arg:     `"0x00112233445566778899aabbccddeeff0011223344"`,
			wantErr: true,
		},
		{
			arg:     `"""`,
			wantErr: true,
		},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			v := &Address{}
			err := v.UnmarshalJSON([]byte(tt.arg))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, *v)
			}
		})
	}
}

func Test_AddressType_Marshal(t *testing.T) {
	tests := []struct {
		arg  Address
		want string
	}{
		{
			arg:  (Address)([AddressLength]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33}),
			want: `"0x00112233445566778899aabbccddeeff00112233"`,
		},
		{
			arg:  Address{},
			want: `"0x0000000000000000000000000000000000000000"`,
		},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			j, err := tt.arg.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(j))
		})
	}
}

func Test_AddressesType_Unmarshal(t *testing.T) {
	tests := []struct {
		arg     string
		want    Addresses
		wantErr bool
	}{
		{
			arg:  `"0x00112233445566778899aabbccddeeff00112233"`,
			want: (Addresses)([]Address{{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33}}),
		},
		{
			arg:  `"00112233445566778899aabbccddeeff00112233"`,
			want: (Addresses)([]Address{{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33}}),
		},
		{
			arg: `["0x00112233445566778899aabbccddeeff00112233", "0x00112233445566778899aabbccddeeff00112233"]`,
			want: (Addresses)([]Address{
				{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33},
				{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33},
			}),
		},
		{
			arg:     `"00112233445566778899aabbccddeeff0011223344"`,
			wantErr: true,
		},
		{
			arg:     `"0x00112233445566778899aabbccddeeff0011223344"`,
			wantErr: true,
		},
		{
			arg:     `"""`,
			wantErr: true,
		},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			v := &Addresses{}
			err := v.UnmarshalJSON([]byte(tt.arg))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, *v)
			}
		})
	}
}

func Test_AddressesType_Marshal(t *testing.T) {
	tests := []struct {
		arg  Addresses
		want string
	}{
		{
			arg:  (Addresses)([]Address{{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33}}),
			want: `["0x00112233445566778899aabbccddeeff00112233"]`,
		},
		{
			arg:  Addresses{},
			want: `[]`,
		},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			j, err := tt.arg.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(j))
		})
	}
}
