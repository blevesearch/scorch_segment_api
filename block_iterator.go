//  Copyright (c) 2026 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package segment

import "github.com/blevesearch/freeway/bitpack"

// PostingsBlockLen is the maximum number of postings a PostingsBlock holds: the
// length of a block of the bit-packed postings, which is that of freeway's codec.
const PostingsBlockLen = bitpack.BlockLen

// PostingsBlock is a batch of postings handed out by a BlockPostingsIterator. Only the
// scalars a scorer needs are carried: doc number, term frequency and field
// norm. There are no locations.
//
// Only the first n entries (n being what the BlockPostingsIterator call returned) are
// meaningful; the remainder is unspecified. Docs are strictly ascending and
// are local to the segment. The array is meant to be allocated once by the
// caller and refilled by every call.
type PostingsBlock struct {
	Docs  [PostingsBlockLen]uint32
	Freqs [PostingsBlockLen]uint32
	// Norms holds the field norm factor (1/sqrt(fieldLength)) of every doc.
	// It is filled only when the iterator was opened with norms.
	Norms [PostingsBlockLen]float32
}

// BlockPostingsList is an optional capability a PostingsList may satisfy to
// let a caller consume its postings a block at a time, rather than one
// posting at a time through a PostingsIterator.
//
// Deletions are taken care of by the iterator: deleted docs never appear in a
// block.
type BlockPostingsList interface {
	// BlockIterator returns an iterator positioned before the first posting.
	// withFreqs / withNorms select which of Freqs / Norms get filled; an
	// unselected array is left unspecified. prealloc may be a iterator
	// previously returned by this method, to be reset and reused (nil is
	// fine).
	BlockIterator(withFreqs, withNorms bool, prealloc BlockPostingsIterator) (BlockPostingsIterator, error)
}

// BlockPostingsIterator iterates over a postings list a block at a time, so that
// whatever sits on top of it pays for an interface call once per batch of
// postings, not once per posting.
type BlockPostingsIterator interface {
	// NextBlock fills b with the next batch of live postings and returns how
	// many it holds. A return of 0 with a nil error means the iterator is
	// exhausted, and it stays exhausted.
	NextBlock(b *PostingsBlock) (n int, err error)

	// SeekBlock fills b with the first batch of live postings whose doc
	// number is >= target, with the entries below target dropped, and
	// returns how many it holds; the entries start at index 0. A return of 0
	// means that nothing at or after target is left. The iterator never moves
	// backwards, so a target below the last returned doc number behaves like
	// NextBlock.
	SeekBlock(target uint64, b *PostingsBlock) (n int, err error)

	// Count is the number of postings of the list, not discounting any
	// deletions. It's cheap.
	Count() uint64

	// LiveCount is the exact number of postings of the list that haven't
	// been deleted. When nothing is deleted it's as cheap as Count; otherwise
	// the list has to be walked, doc numbers only, so use it when the count is
	// all that's needed. It's independent of the iterator's position, and
	// doesn't move it.
	LiveCount() (uint64, error)

	// BytesRead is the number of bytes read from disk so far by this iterator,
	// cumulatively.
	BytesRead() uint64
}

// BlockBounds describes a block of postings by what bounds the score of
// every posting in it: the highest term frequency, and the highest norm factor
// (the shortest field). The score of a posting rises with both, so the score
// of the pair is an upper bound on the scores of the block. The two need not
// come from the same posting.
type BlockBounds struct {
	// LastDoc is the highest doc number of the block (math.MaxUint32 if there
	// is no such block).
	LastDoc uint32

	// MaxFreq is the highest term frequency of the block. It is only
	// meaningful if FreqBounded.
	MaxFreq uint32
	// FreqBounded is false if the frequencies of the block aren't bounded: a
	// format may only record them up to a cap.
	FreqBounded bool

	// MaxNorm is the highest norm factor (1/sqrt(field length)) of the block;
	// +Inf if it isn't known.
	MaxNorm float32
}

// BlockMaxPostingsIterator is an optional capability of a BlockPostingsIterator: bounds on the
// scores of the blocks of the list. It is what block-max pruning (WAND) is made
// of.
type BlockMaxPostingsIterator interface {
	// BoundsAt are the bounds of the first block whose last doc is >= target,
	// and whether there is such a block. It only looks at the list's skip
	// data: nothing is decoded and the iterator doesn't move, so it can be
	// asked about any target, in any order, whenever a pruner wants a bound
	// without committing to read from there.
	BoundsAt(target uint32) (BlockBounds, bool)

	// DecodedBounds are the bounds of the block that the last NextBlock or
	// SeekBlock decoded -- the block the postings it returned are from.
	DecodedBounds() BlockBounds

	// TermBounds are the bounds of the whole list: the highest frequency and
	// norm factor over its blocks. LastDoc is that of the last block.
	TermBounds() BlockBounds
}
