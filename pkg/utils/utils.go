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
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	MAX_CN_LENGTH = 64 // max commonName length for spec.commonName field (cert manager etcd credentials)
)

// IPAtOffset calculates an IP address within a CIDR range at a given offset.
func IPAtOffset(cidr string, offset uint32) (net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	ip = ip.To4()
	if ip == nil {
		return nil, fmt.Errorf("only IPv4 is supported")
	}

	network := ipnet.IP.To4()

	base := uint32(network[0])<<24 |
		uint32(network[1])<<16 |
		uint32(network[2])<<8 |
		uint32(network[3])

	target := base + offset

	out := net.IPv4(
		byte(target>>24),
		byte(target>>16),
		byte(target>>8),
		byte(target),
	)

	if !ipnet.Contains(out) {
		return nil, fmt.Errorf("offset %d out of range for %s", offset, cidr)
	}

	return out, nil
}

// MergeStringMap merges a source map into a destination map, overwriting existing keys.
func MergeStringMap(dst, src map[string]string) map[string]string {
	if dst == nil && src == nil {
		return nil
	}
	if dst == nil {
		dst = map[string]string{}
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// PortString converts an int32 port number to its string representation.
func PortString(p int32) string {
	return strconv.Itoa(int(p))
}

// IntstrFromInt converts an integer port to an IntOrString type.
func IntstrFromInt(port int32) intstr.IntOrString {
	return intstr.IntOrString{Type: intstr.Int, IntVal: port}
}

// GetMajorMinorString extracts the major and minor version from a full version string.
func GetMajorMinorString(version string) string {
	if len(version) < 2 {
		return ""
	}

	v := version[1:]

	firstDot := strings.IndexByte(v, '.')
	if firstDot == -1 {
		return ""
	}

	secondDot := strings.IndexByte(v[firstDot+1:], '.')
	if secondDot == -1 {
		return ""
	}

	secondDot += firstDot + 1

	return v[:secondDot]
}

// Serialize k8s object to yaml
func GetObjYaml(obj runtime.Object) string {
	s := json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		scheme.Scheme, scheme.Scheme,
		json.SerializerOptions{Yaml: true})

	b := new(bytes.Buffer)

	err := s.Encode(obj, b)
	// should never happen
	if err != nil {
		panic("unexpected error: " + err.Error())
	}

	return b.String()
}

// GetObjJSON serializes a Kubernetes runtime object to a JSON string.
func GetObjJSON(obj runtime.Object) string {
	s := json.NewSerializerWithOptions(
		json.DefaultMetaFactory,
		scheme.Scheme,
		scheme.Scheme,
		json.SerializerOptions{
			Yaml:   false,
			Pretty: true,
			Strict: false,
		},
	)

	b := new(bytes.Buffer)

	err := s.Encode(obj, b)
	// should never happen
	if err != nil {
		panic("unexpected error: " + err.Error())
	}

	return b.String()
}

func TruncateToMaxLength(podName string) (string, bool) {
	if len(podName) > MAX_CN_LENGTH {
		return podName[:64], true
	}
	return podName, false
}
