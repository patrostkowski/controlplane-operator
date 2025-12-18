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

package rbac

import (
	"crypto/rand"
	"fmt"
)

const (
	tokenIDLen     = 6
	tokenSecretLen = 16
	tokenAlphabet  = "abcdefghijklmnopqrstuvwxyz0123456789"
)

type BootstrapToken struct {
	ID     string
	Secret string
}

func NewBootstrapToken() (BootstrapToken, error) {
	id, err := randString(tokenIDLen)
	if err != nil {
		return BootstrapToken{}, err
	}
	sec, err := randString(tokenSecretLen)
	if err != nil {
		return BootstrapToken{}, err
	}
	return BootstrapToken{ID: id, Secret: sec}, nil
}

func randString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = tokenAlphabet[int(b[i])%len(tokenAlphabet)]
	}
	return string(out), nil
}

func (t BootstrapToken) String() string {
	return fmt.Sprintf("%s.%s", t.ID, t.Secret)
}
