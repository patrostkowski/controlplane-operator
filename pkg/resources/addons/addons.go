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

package addons

import (
	"github.com/patrostkowski/controlplane-operator/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addonsBuilder struct {
	cc cluster.AddonSpec
}

func NewAddonsBuilder(cc cluster.AddonSpec) cluster.ObjectProducer {
	return addonsBuilder{cc: cc}
}

// Objects implements cluster.ObjectProducer
func (e addonsBuilder) Objects() []client.Object {
	var objs []client.Object

	objs = append(objs, e.buildKubeproxy()...)
	objs = append(objs, e.buildFlannel()...)
	objs = append(objs, e.buildCoreDNS()...)
	objs = append(objs, e.buildCSI()...)
	objs = append(objs, e.buildKonnectivityAgent()...)

	return objs
}
