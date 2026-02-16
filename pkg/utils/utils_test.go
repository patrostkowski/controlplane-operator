// Copyright 2025 Patryk Rostkowski
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestIPAtOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cidr    string
		offset  uint32
		want    string
		wantErr bool
	}{
		{"basic", "10.0.0.0/24", 1, "10.0.0.1", false},
		{"networkAddr", "192.168.1.0/24", 0, "192.168.1.0", false},
		{"lastInRange", "192.168.1.0/30", 3, "192.168.1.3", false},
		{"outOfRange", "192.168.1.0/30", 4, "", true},
		{"badCIDR", "not-a-cidr", 1, "", true},
		{"ipv6NotSupported", "2001:db8::/64", 1, "", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ip, err := IPAtOffset(tt.cidr, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if ip.String() != tt.want {
				t.Fatalf("got %q want %q", ip.String(), tt.want)
			}
		})
	}
}

func TestMergeStringMap(t *testing.T) {
	t.Parallel()

	t.Run("bothNil", func(t *testing.T) {
		t.Parallel()
		if got := MergeStringMap(nil, nil); got != nil {
			t.Fatalf("expected nil, got %#v", got)
		}
	})

	t.Run("dstNil", func(t *testing.T) {
		t.Parallel()
		got := MergeStringMap(nil, map[string]string{"a": "1"})
		if got == nil || got["a"] != "1" {
			t.Fatalf("unexpected result: %#v", got)
		}
	})

	t.Run("srcNil", func(t *testing.T) {
		t.Parallel()
		dst := map[string]string{"a": "1"}
		got := MergeStringMap(dst, nil)
		if got["a"] != "1" {
			t.Fatalf("unexpected result: %#v", got)
		}
	})

	t.Run("overwrites", func(t *testing.T) {
		t.Parallel()
		dst := map[string]string{"a": "1", "b": "2"}
		src := map[string]string{"b": "22", "c": "3"}
		got := MergeStringMap(dst, src)
		if got["a"] != "1" || got["b"] != "22" || got["c"] != "3" {
			t.Fatalf("unexpected merge: %#v", got)
		}
	})
}

func TestPortString(t *testing.T) {
	t.Parallel()

	if got := PortString(80); got != "80" {
		t.Fatalf("got %q want %q", got, "80")
	}
	if got := PortString(0); got != "0" {
		t.Fatalf("got %q want %q", got, "0")
	}
}

func TestIntstrFromInt(t *testing.T) {
	t.Parallel()

	got := IntstrFromInt(8080)
	if got.Type != intstr.Int || got.IntValue() != 8080 {
		t.Fatalf("unexpected IntOrString: %#v", got)
	}
}
