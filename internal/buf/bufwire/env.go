// Copyright 2020 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bufwire

import (
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
)

type env struct {
	image  bufimage.Image
	config *bufconfig.Config
}

func newEnv(image bufimage.Image, config *bufconfig.Config) *env {
	return &env{
		image:  image,
		config: config,
	}
}

func (e *env) Image() bufimage.Image {
	return e.image
}

func (e *env) Config() *bufconfig.Config {
	return e.config
}
