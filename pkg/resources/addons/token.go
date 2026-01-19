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
	"fmt"
	"strings"

	bootstraphandle "k8s.io/cluster-bootstrap/token/util"
)

type BootstrapToken struct {
	ID     string
	Secret string
}

func NewBootstrapToken() (BootstrapToken, error) {
	tok, err := bootstraphandle.GenerateBootstrapToken()
	if err != nil {
		return BootstrapToken{}, err
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 2 {
		return BootstrapToken{}, fmt.Errorf("unexpected token format: %q", tok)
	}
	return BootstrapToken{ID: parts[0], Secret: parts[1]}, nil
}

func (t BootstrapToken) String() string {
	return fmt.Sprintf("%s.%s", t.ID, t.Secret)
}
