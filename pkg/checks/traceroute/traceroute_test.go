package traceroute

import (
	"net"
	"reflect"
	"testing"
)

func TestHopAddress_String(t *testing.T) {
	type fields struct {
		IP   string
		Port int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "No Port", fields: fields{IP: "100.1.1.7"}, want: "100.1.1.7"},
		{name: "With Port", fields: fields{IP: "100.1.1.7", Port: 80}, want: "100.1.1.7:80"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := HopAddress{
				IP:   tt.fields.IP,
				Port: tt.fields.Port,
			}
			if got := a.String(); got != tt.want {
				t.Errorf("HopAddress.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newHopAddress(t *testing.T) {
	type args struct {
		addr net.Addr
	}
	tests := []struct {
		name string
		args args
		want HopAddress
	}{
		{
			name: "Works with TCP",
			args: args{
				addr: &net.TCPAddr{IP: net.ParseIP("100.1.1.7"), Port: 80},
			},
			want: HopAddress{
				IP:   "100.1.1.7",
				Port: 80,
			},
		},
		{
			name: "Works with UDP",
			args: args{
				addr: &net.UDPAddr{IP: net.ParseIP("100.1.1.7"), Port: 80},
			},
			want: HopAddress{
				IP:   "100.1.1.7",
				Port: 80,
			},
		},
		{
			name: "Works with IP",
			args: args{
				addr: &net.IPAddr{IP: net.ParseIP("100.1.1.7")},
			},
			want: HopAddress{
				IP: "100.1.1.7",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newHopAddress(tt.args.addr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newHopAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
