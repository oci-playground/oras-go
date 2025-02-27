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

package content

import (
	"context"
	"encoding/json"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	artifactspec "github.com/oras-project/artifacts-spec/specs-go/v1"
	"oras.land/oras-go/v2/internal/descriptor"
	"oras.land/oras-go/v2/internal/docker"
)

// PredecessorFinder finds out the nodes directly pointing to a given node of a
// directed acyclic graph.
// In other words, returns the "parents" of the current descriptor.
// PredecessorFinder is an extension of Storage.
type PredecessorFinder interface {
	// Predecessors returns the nodes directly pointing to the current node.
	Predecessors(ctx context.Context, node ocispec.Descriptor) ([]ocispec.Descriptor, error)
}

// GraphStorage represents a CAS that supports direct predecessor node finding.
type GraphStorage interface {
	Storage
	PredecessorFinder
}

// ReadOnlyGraphStorage represents a read-only GraphStorage.
type ReadOnlyGraphStorage interface {
	ReadOnlyStorage
	PredecessorFinder
}

// Successors returns the nodes directly pointed by the current node.
// In other words, returns the "children" of the current descriptor.
func Successors(ctx context.Context, fetcher Fetcher, node ocispec.Descriptor) ([]ocispec.Descriptor, error) {
	switch node.MediaType {
	case docker.MediaTypeManifest, ocispec.MediaTypeImageManifest:
		content, err := FetchAll(ctx, fetcher, node)
		if err != nil {
			return nil, err
		}

		// docker manifest and oci manifest are equivalent for successors.
		var manifest ocispec.Manifest
		if err := json.Unmarshal(content, &manifest); err != nil {
			return nil, err
		}
		return append([]ocispec.Descriptor{manifest.Config}, manifest.Layers...), nil
	case docker.MediaTypeManifestList, ocispec.MediaTypeImageIndex:
		content, err := FetchAll(ctx, fetcher, node)
		if err != nil {
			return nil, err
		}

		// docker manifest list and oci index are equivalent for successors.
		var index ocispec.Index
		if err := json.Unmarshal(content, &index); err != nil {
			return nil, err
		}
		return index.Manifests, nil
	case artifactspec.MediaTypeArtifactManifest: // TODO: deprecate
		content, err := FetchAll(ctx, fetcher, node)
		if err != nil {
			return nil, err
		}

		var manifest artifactspec.Manifest
		if err := json.Unmarshal(content, &manifest); err != nil {
			return nil, err
		}
		var nodes []ocispec.Descriptor
		if manifest.Subject != nil {
			nodes = append(nodes, descriptor.ArtifactToOCI(*manifest.Subject))
		}
		for _, blob := range manifest.Blobs {
			nodes = append(nodes, descriptor.ArtifactToOCI(blob))
		}
		return nodes, nil
	case ocispec.MediaTypeArtifactManifest:
		content, err := FetchAll(ctx, fetcher, node)
		if err != nil {
			return nil, err
		}

		var manifest ocispec.Artifact
		if err := json.Unmarshal(content, &manifest); err != nil {
			return nil, err
		}
		var nodes []ocispec.Descriptor
		if manifest.Subject != nil {
			nodes = append(nodes, *manifest.Subject)
		}
		return append(nodes, manifest.Blobs...), nil
	}
	return nil, nil
}
