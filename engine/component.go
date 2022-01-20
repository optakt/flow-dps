// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package engine

import (
	"time"

	"github.com/rs/zerolog"
)

// Component wraps any engine unit that can be started and stopped.
type component struct {
	log  zerolog.Logger
	run  func() error
	stop func()
}

// Run launches the component.
func (c *component) Run(notify chan error) {
	start := time.Now()

	c.log.Info().Msg("component starting")
	err := c.run()
	if err != nil {
		c.log.Error().Err(err).Msg("component failed")
		notify <- err
		return
	}

	notify <- nil

	duration := time.Now().Sub(start)
	c.log.Info().
		Str("duration", duration.Round(time.Second).String()).
		Msg("component done")
}

// Stop stops the component.
func (c *component) Stop() {
	c.stop()
	c.log.Info().Msg("component stopped")
}
