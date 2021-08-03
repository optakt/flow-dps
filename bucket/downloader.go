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

package bucket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/fvm/errors"
)

type Downloader struct {
	logger zerolog.Logger

	// bucket is the name of the S3 bucket to use.
	bucket string

	// service is used to regularly poll the AWS API to check for new items in the bucket.
	service *s3.S3

	// downloader is used to download bucket items into the filesystem.
	downloader *s3manager.Downloader

	outputDir string
	wg        sync.WaitGroup
	stopCh    chan struct{}
}

func NewDownloader(logger zerolog.Logger, region, bucket, outputDir string) (*Downloader, error) {
	_, err := os.Stat(outputDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("download directory path (%s) does not exist: %w", outputDir, err)
	}

	cfg := &aws.Config{
		Region: aws.String(region),
	}

	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not initialize AWS session: %w", err)
	}

	d := Downloader{
		logger:     logger,
		bucket:     bucket,
		service:    s3.New(sess),
		downloader: s3manager.NewDownloader(sess),
		outputDir:  outputDir,
		stopCh:     make(chan struct{}),
	}

	return &d, nil
}

func (d *Downloader) Run() {
	d.wg.Add(1)
	for {
		// Check if we should stop.
		select {
		case <-d.stopCh:
			return
		default:
		}

		// List bucket items.
		req := &s3.ListObjectsV2Input{Bucket: aws.String(d.bucket)}
		resp, err := d.service.ListObjectsV2(req)
		if err != nil {
			// FIXME: Handle error properly
			panic(err)
		}

		// Look at each bucket item to check whether or not it has already been downloaded. If not, download it.
		var totalDownloadedBytes, totalDownloadedFiles int64
		for _, item := range resp.Contents {
			path := filepath.Join(d.outputDir, *item.Key)

			_, err := os.Stat(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				// FIXME: Handle error properly
				panic(err)
			}

			// File already downloaded, can be skipped.
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			file, err := os.Create(path)
			if err != nil {
				// FIXME: Handle error properly
				panic(err)
			}

			req := &s3.GetObjectInput{
				Bucket: aws.String(d.bucket),
				Key:    aws.String(*item.Key),
			}
			n, err := d.downloader.Download(file, req)
			if err != nil {
				// FIXME: Handle error properly
				panic(err)
			}

			totalDownloadedBytes += n
			totalDownloadedFiles++
		}

		if totalDownloadedFiles > 0 {
			d.logger.Debug().
				Str("bucket", d.bucket).
				Int64("download_bytes", totalDownloadedBytes).
				Int64("download_files", totalDownloadedFiles).
				Msg("successfully downloaded files from bucket")
		}

		// FIXME: Seems necessary to avoid spamming the S3 API.
		//		  Set to a low value for now for testing.
		// 		  How often should we poll realistically?
		// 		  Should this be configurable?
		time.Sleep(1 * time.Second)
	}
}

func (d *Downloader) Stop(ctx context.Context) error {
	close(d.stopCh)
	d.wg.Wait()
	return nil
}
