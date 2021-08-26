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

package gcp

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

const cacheSize = 1024

type Downloader struct {
	bucket *storage.BucketHandle

	cache        map[string]time.Time
	notify       chan string
	lastConsumed time.Time

	stop chan struct{}
}

// NewDownloader creates a Downloader instance that has access to a GCP bucket.
func NewDownloader(bucket *storage.BucketHandle) *Downloader {
	d := Downloader{
		bucket: bucket,
		cache:  make(map[string]time.Time, cacheSize),
		notify: make(chan string, cacheSize),
		stop:   make(chan struct{}),
	}

	return &d
}

// Run launches a polling process which continuously caches files to be downloaded ordered by creation time.
func (d *Downloader) Run() error {
	for {
		select {
		case <-d.stop:
			return nil
		default:
		}

		// If the cache is already full, wait for one item to be consumed before loading more into the cache.
		if len(d.cache) == cacheSize {
			time.Sleep(1 * time.Second)
			continue
		}

		// Make a query to only populate the name and creation time of each bucket item.
		query := &storage.Query{}
		err := query.SetAttrSelection([]string{"Name", "Created"})
		if err != nil {
			return fmt.Errorf("could not create Google Cloud Storage query: %w", err)
		}

		it := d.bucket.Objects(context.TODO(), query)
		var files []*storage.ObjectAttrs
		for {
			file, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return fmt.Errorf("could not iterate through Google Cloud Storage files: %w", err)
			}
			files = append(files, file)
		}

		// Sort files by creation time.
		sort.Slice(files, func(i, j int) bool {
			return files[i].Created.Before(files[j].Created)
		})

		for _, file := range files {
			// Ignore files that were already consumed.
			if file.Created.Before(d.lastConsumed) {
				continue
			}

			// Stop adding new files if the cache is already full.
			if len(d.cache) == cacheSize {
				break
			}
			d.cache[file.Name] = file.Created
			d.notify <- file.Name
		}
	}
}

// Stop stops the Downloader's polling process.
func (d *Downloader) Stop() {
	close(d.stop)
	close(d.notify)
}

func (d *Downloader) Notify() <-chan string {
	return d.notify
}

// Read reads and returns the contents of the next available segment.
func (d *Downloader) Read(name string) ([]byte, error) {
	_, cached := d.cache[name]
	if !cached {
		return nil, fmt.Errorf("missing item from downloader cache: %s", name)
	}

	block := d.bucket.Object(name)
	r, err := block.NewReader(context.TODO()) // FIXME: Use a proper context
	if err != nil {
		return nil, fmt.Errorf("could not create bucket item reader: %w", err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("could not read bucket item: %w", err)
	}

	d.lastConsumed = d.cache[name]
	delete(d.cache, name)

	return b, nil
}
