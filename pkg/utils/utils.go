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
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func EnsureCreatedAndOwned(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	owner client.Object,
	template client.Object,
	log logr.Logger,
	mutate func(obj client.Object) error,
) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		obj := template.DeepCopyObject().(client.Object)
		_, err := controllerutil.CreateOrUpdate(ctx, c, obj, func() error {
			if err := controllerutil.SetControllerReference(owner, obj, scheme); err != nil {
				return err
			}
			if mutate != nil {
				return mutate(obj)
			}
			return nil
		})
		return err
	})
}

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

func PortString(p int32) string {
	return strconv.Itoa(int(p))
}

func IntstrFromInt(port int32) intstr.IntOrString {
	return intstr.IntOrString{Type: intstr.Int, IntVal: port}
}

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
