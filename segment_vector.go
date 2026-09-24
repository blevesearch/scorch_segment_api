//  Copyright (c) 2023 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build vectors
// +build vectors

package segment

import (
	"encoding/json"

	"github.com/RoaringBitmap/roaring/v2"
	index "github.com/blevesearch/bleve_index_api"
)

type VecPostingsList interface {
	DiskStatsReporter

	Iterator(prealloc VecPostingsIterator) VecPostingsIterator

	Size() int

	Count() uint64

	// NOTE deferred for future work

	// And(other PostingsList) PostingsList
	// Or(other PostingsList) PostingsList
}

type VecPostingsIterator interface {
	DiskStatsReporter

	// The caller is responsible for copying whatever it needs from
	// the returned Posting instance before calling Next(), as some
	// implementations may return a shared instance to reduce memory
	// allocations.
	Next() (VecPosting, error)

	// Advance will return the posting with the specified doc number
	// or if there is no such posting, the next posting.
	// Callers MUST NOT attempt to pass a docNum that is less than or
	// equal to the currently visited posting doc Num.
	Advance(docNum uint64) (VecPosting, error)

	Size() int
}

type VectorIndex interface {
	// Search performs a kNN search for the given query vector and returns a postings list.
	// - qVector: the query vector
	// - k: the number of similar vectors to return
	// - params: additional search parameters
	Search(qVector []float32, k int64, params json.RawMessage) (VecPostingsList, error)
	// SearchWithFilter performs a kNN search for the given query vector, filtering results based on eligible documents
	// - qVector: the query vector
	// - k: the number of similar vectors to return
	// - eligibleList: list of eligible documents to consider
	// - params: additional search parameters
	SearchWithFilter(qVector []float32, k int64, eligibleList index.EligibleDocumentList, params json.RawMessage) (VecPostingsList, error)
	// Close releases any resources held by the VectorIndex.
	Close()
	Size() uint64

	ObtainKCentroidCardinalitiesFromIVFIndex(limit int, descending bool) ([]index.CentroidCardinality, error)
}

// PreassignedCentroids is the result of a phase-one coarse quantizer search:
// the clusters of an IVF vector index ranked by their distance from a query
// vector.
//
// It exists so that a query against many segments that were all built from a
// single trained index does not have to repeat the same coarse quantizer
// search once per segment. Those segments share a centroid layout by
// construction - the fast merge path clones the trained index's quantizer into
// every segment it writes - so cluster number i covers the same region of the
// vector space in all of them, and one ranking is valid for all of them.
//
// Phase two then feeds this ranking to each segment, which probes exactly
// those clusters instead of deriving them itself.
type PreassignedCentroids struct {
	// Source is the vector index whose coarse quantizer produced IDs and
	// Distances. It is opaque to the caller: a segment implementation uses it
	// to confirm that its own clusters are numbered the same way before
	// trusting the ranking. Segments that were never built from this trained
	// index - typically small ones that missed the fast merge path and were
	// clustered independently - will not match, and must fall back to an
	// ordinary search.
	Source interface{}

	// IDs holds every centroid id of the producing index, ordered by
	// increasing distance from the query vector.
	IDs []int64

	// Distances holds the distance from the query vector to each centroid in
	// IDs, in the same order. For a binary index these are Hamming distances
	// widened to float32.
	Distances []float32

	// Nprobe is how many leading entries of IDs a search should probe by
	// default, taken from the producing index. A search is free to widen
	// beyond it, up to len(IDs), when it needs more candidates.
	Nprobe int
}

// Valid reports whether p carries a usable centroid ranking.
func (p *PreassignedCentroids) Valid() bool {
	return p != nil && p.Source != nil && len(p.IDs) > 0 &&
		len(p.Distances) >= len(p.IDs)
}

type TrainedSegment interface {
	Segment
	GetCoarseQuantizer(field string) (interface{}, error)

	// SearchCentroids runs a coarse quantizer search for qVector against this
	// segment's trained index for the given field, ranking every centroid by
	// its distance from qVector. This is phase one of a two phase vector
	// search; the result is handed to PreassignedVectorIndex implementations
	// for phase two.
	//
	// It returns a nil result, and no error, when the field has no trained IVF
	// index to search - in which case callers simply run an ordinary per
	// segment search.
	SearchCentroids(field string, qVector []float32) (*PreassignedCentroids, error)
}

// PreassignedVectorIndex is implemented by a VectorIndex that can skip its own
// coarse quantizer search and instead probe a set of clusters chosen upfront
// by whoever produced a PreassignedCentroids.
//
// Implementations must verify the layout themselves rather than trusting the
// caller: a segment whose clusters are numbered differently would otherwise
// return results for the wrong regions of the vector space. Both search
// methods therefore fall back to their ordinary equivalents whenever the
// layout does not match, so a caller can pass pre unconditionally.
type PreassignedVectorIndex interface {
	VectorIndex

	// SharesCentroidLayout reports whether this index's clusters are numbered
	// identically to those of the index that produced pre, which is the
	// precondition for the two search methods below taking their fast path.
	SharesCentroidLayout(pre *PreassignedCentroids) (bool, error)

	// SearchPreassigned is Search, but probing the clusters named by pre
	// instead of running a coarse quantizer search of its own.
	SearchPreassigned(qVector []float32, k int64, pre *PreassignedCentroids,
		params json.RawMessage) (VecPostingsList, error)

	// SearchWithFilterPreassigned is SearchWithFilter, but ranking the
	// clusters that hold eligible vectors by reusing pre's ranking instead of
	// running a coarse quantizer search of its own.
	SearchWithFilterPreassigned(qVector []float32, k int64,
		eligibleList index.EligibleDocumentList, pre *PreassignedCentroids,
		params json.RawMessage) (VecPostingsList, error)
}

type VectorSegment interface {
	Segment
	InterpretVectorIndex(field string, except *roaring.Bitmap) (VectorIndex, error)
}

type VecPosting interface {
	Number() uint64

	Score() float32

	Size() int
}
