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
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rs/zerolog"
)

type Downloader struct {
	logger zerolog.Logger

	// bucket is the name of the S3 bucket to use.
	bucket string

	// service is used to regularly poll the AWS API to check for new items in the bucket.
	service *s3.S3

	// downloader is used to download bucket items into the filesystem.
	downloader *s3manager.Downloader

	// buffer stores the bytes of the current segment until it is read, at which point it gets reset.
	buffer *aws.WriteAtBuffer

	// segment holds a copy of the bytes from buffer, but gets dynamically truncated as it is read.
	segment []byte

	// the index of the next segment to retrieve.
	index int

	wg     sync.WaitGroup
	readCh chan struct{}
	stopCh chan struct{}
}

// NewDownloader creates a new AWS S3 session for downloading and returns a Downloader instance that uses it.
func NewDownloader(logger zerolog.Logger, region, bucket string) (*Downloader, error) {
	cfg := &aws.Config{
		Region: aws.String(region),
	}

	// Create the S3 session used to list bucket items and download them.
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not initialize AWS session: %w", err)
	}

	d := Downloader{
		logger:     logger,
		bucket:     bucket,
		service:    s3.New(sess),
		downloader: s3manager.NewDownloader(sess),
		buffer:     &aws.WriteAtBuffer{},
		stopCh:     make(chan struct{}),
		readCh:     make(chan struct{}),
	}

	return &d, nil
}

// Read reads into the given byte slice as long as it has available bytes to read from the segments that have been
// downloaded.
// FIXME: Make this non-blocking and timeout after a while, returning an io.EOF error for the LiveReader. That might
// 		  trigger a corruption error on the Prometheus Live Reader side, which would not be great. Need to look into it.
func (d *Downloader) Read(p []byte) (int, error) {
	// If we don't have a segment yet, wait until one is available.
	for len(d.segment) == 0 {
		// Check if we should stop.
		select {
		case <-d.stopCh:
			return 0, nil
		default:
		}

		time.Sleep(1 * time.Second)
		d.segment = d.buffer.Bytes()
	}

	// If the requested bytes to read is greater than the available number of bytes,
	// only read what is available for now.
	// TODO: Could be improved by appending the next segment instead of resetting the buffer whenever we reach the end,
	//		 but this is not necessary since the LiveReader handles incomplete reads with its `permissive` attribute and
	//		 could also be problematic depending on how the S3 SDK writes on the buffer with WriteAt. We might confuse its
	//       index my dynamically truncating the buffer it's writing in.
	//       See https://github.com/prometheus/prometheus/blob/main/tsdb/wal/live_reader.go#L59-L61
	n := min(len(p), len(d.segment))

	// Pick only the requested length of bytes.
	readBytes := d.segment[:n]

	// Remove those bytes from the buffer.
	d.segment = d.segment[n:]

	// Reset the buffer if it has been read completely.
	if len(d.segment) == 0 {
		d.buffer = &aws.WriteAtBuffer{}
	}

	// Increase index to make the `Run` routine retrieve the next segment.
	d.index++

	// Check if readCh is still open. If it has been closed, the downloader is stopping
	// and thus we should just return asap.
	_, ok := <-d.readCh
	if !ok {
		return 0, nil
	}

	// Notify the `Run` routine that it can retrieve the next segment now.
	d.readCh <- struct{}{}

	copy(p, readBytes)

	return n, nil
}

// Run is designed to run in the background and download the next segment whenever one has been fully read.
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
			d.logger.Error().Err(err).Msg("could not list bucket items")
			continue
		}

		// Segments are formatted like: 00000000, 00000001, etc.
		wantKey := fmt.Sprintf("%08d", d.index)

		// Look at each bucket item to check whether or not it has already been downloaded. If not, download it.
		var totalDownloadedBytes int64
		for _, item := range resp.Contents {
			// Skip any item that does not match the index we are looking for.
			if *item.Key != wantKey {
				continue
			}

			// Download the segment into the buffer.
			req := &s3.GetObjectInput{
				Bucket: aws.String(d.bucket),
				Key:    aws.String(*item.Key),
			}
			n, err := d.downloader.Download(d.buffer, req)
			if err != nil {
				d.logger.Error().Err(err).Msg("could not download bucket item")
				continue
			}

			totalDownloadedBytes += n
		}

		if totalDownloadedBytes == 0 {
			d.logger.Debug().
				Str("bucket", d.bucket).
				Str("segment_key", wantKey).
				Int64("download_bytes", totalDownloadedBytes).
				Msg("waiting for segment to be available")

			// FIXME: Seems necessary to avoid spamming the S3 API.
			//		  Set to a low value for now for testing.
			// 		  How often should we poll realistically?
			// 		  Should this be configurable?
			time.Sleep(1 * time.Second)
		} else {
			d.logger.Debug().
				Str("bucket", d.bucket).
				Str("segment_key", wantKey).
				Int64("download_bytes", totalDownloadedBytes).
				Msg("successfully downloaded segment from bucket")

			// Wait until segment has been read by the feeder before downloading the next one.
			<-d.readCh
		}
	}
}

// Stop makes the Run and Read methods stop gracefully.
func (d *Downloader) Stop(ctx context.Context) error {
	close(d.stopCh)
	close(d.readCh)
	d.wg.Wait()
	return nil
}

// TODO: Remove this once generics are there and math.Min handles integers generically.
func min(i, j int) int {
	if i < j {
		return i
	}
	return j
}
