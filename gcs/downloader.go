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

package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type Downloader struct {
	bucket *storage.BucketHandle
}

// NewDownloader creates a Poller instance that has access to a GCS bucket.
func NewDownloader(bucket *storage.BucketHandle) *Downloader {
	d := Downloader{
		bucket: bucket,
	}

	return &d
}

// List returns a list of available files to be downloaded.
func (d *Downloader) List(ctx context.Context, prefix string) ([]string, error) {
	it := d.bucket.Objects(ctx, &storage.Query{
		Prefix:    prefix,
	})

	var files []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		files = append(files, attrs.Name)
	}

	return files, nil
}

// Download downloads a file from the bucket to the given destination folder.
func (d *Downloader) Download(ctx context.Context, destination, source string) error {
	dir := filepath.Dir(destination)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not create destination directory: %w", err)
	}

	download, err := d.bucket.Object(source).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("could not create GCS object reader: %w", err)
	}
	defer download.Close()

	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, download)
	if err != nil {
		return fmt.Errorf("could not download file: %w", err)
	}
	return nil
}