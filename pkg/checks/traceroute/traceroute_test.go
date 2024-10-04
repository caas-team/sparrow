// sparrow
// (C) 2024, Deutsche Telekom IT GmbH
//
// Deutsche Telekom IT GmbH and all other contributors /
// copyright owners license this file to you under the Apache
// License, Version 2.0 (the "License"); you may not use this
// file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
