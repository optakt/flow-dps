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

package network

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/follower"
	"github.com/onflow/flow-go/model/bootstrap"

	"github.com/optakt/flow-dps/gcs"
)

// FIXME: Should this be configurable?
const outputDir = "./bootstrap"

func RetrieveIdentities(log zerolog.Logger, downloader *gcs.Downloader, token string) ([]follower.BootstrapNodeInfo, error) {
	prefix := fmt.Sprintf("%s/%s/", token, bootstrap.DirnamePublicBootstrap)
	files, err := downloader.List(context.Background(), prefix)
	if err != nil {
		return nil, fmt.Errorf("could not get list of files from GCS")
	}

	log.Info().Msgf("found %d files in public bootstrap GCS Bucket", len(files))

	// download found files
	for _, file := range files {
		fullPath := filepath.Join(outputDir, "public-root-information", filepath.Base(file))
		log.Info().Str("source", file).Str("destination", fullPath).Msgf("downloading file from transit servers")

		err = downloader.Download(context.Background(), fullPath, file)
		if err != nil {
			return nil, fmt.Errorf("could not download file from GCS")
		}
	}

	// FIXME: Parse those files to generate Bootstrap Identities from them.

	return nil, nil
}