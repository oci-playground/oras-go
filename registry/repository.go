/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package registry

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

// Repository is an ORAS target and an union of the blob and the manifest CASs.
// As specified by https://docs.docker.com/registry/spec/api/, it is natural to
// assume that content.Resolver interface only works for manifests. Tagging a
// blob may be resulted in an `ErrUnsupported` error. However, this interface
// does not restrict tagging blobs.
// Since a repository is an union of the blob and the manifest CASs, all
// operations defined in the `BlobStore` are executed depending on the media
// type of the given descriptor accordingly.
// Furthermore, this interface also provides the ability to enforce the
// separation of the blob and the manifests CASs.
type Repository interface {
	content.Storage
	content.Deleter
	content.TagResolver
	ReferenceFetcher
	ReferencePusher

	// Blobs provides access to the blob CAS only, which contains config blobs,
	// layers, and other generic blobs.
	Blobs() BlobStore

	// Manifests provides access to the manifest CAS only.
	Manifests() ManifestStore

	// Tags lists the tags available in the repository.
	// Since the returned tag list may be paginated by the underlying
	// implementation, a function should be passed in to process the paginated
	// tag list.
	// `last` argument is the `last` parameter when invoking the tags API.
	// If `last` is NOT empty, the entries in the response start after the
	// tag specified by `last`. Otherwise, the response starts from the top
	// of the Tags list.
	// Note: When implemented by a remote registry, the tags API is called.
	// However, not all registries supports pagination or conforms the
	// specification.
	// References:
	// - https://github.com/opencontainers/distribution-spec/blob/main/spec.md#content-discovery
	// - https://docs.docker.com/registry/spec/api/#tags
	// See also `Tags()` in this package.
	Tags(ctx context.Context, last string, fn func(tags []string) error) error
}

// BlobStore is a CAS with the ability to stat and delete its content.
type BlobStore interface {
	content.Storage
	content.Deleter
	content.Resolver
	ReferenceFetcher
}

// ManifestStore is a CAS with the ability to stat and delete its content.
// Besides, ManifestStore provides reference tagging.
type ManifestStore interface {
	BlobStore
	content.Tagger
	ReferencePusher
}

// ReferencePusher provides advanced push with the tag service.
type ReferencePusher interface {
	// PushReference pushes the manifest with a reference tag.
	PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error
}

// ReferenceFetcher provides advanced fetch with the tag service.
type ReferenceFetcher interface {
	// FetchReference fetches the content identified by the reference.
	FetchReference(ctx context.Context, reference string) (ocispec.Descriptor, io.ReadCloser, error)
}

// ReferrerFinder provides the Referrers API.
// Reference: https://github.com/oras-project/artifacts-spec/blob/main/manifest-referrers-api.md
type ReferrerFinder interface {
	Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error
}

// Tags lists the tags available in the repository.
func Tags(ctx context.Context, repo Repository) ([]string, error) {
	var res []string
	if err := repo.Tags(ctx, "", func(tags []string) error {
		res = append(res, tags...)
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}
