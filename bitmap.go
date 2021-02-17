package segment

import "io"

type IntIterable interface {
	HasNext() bool
	Next() uint32
}

type IntPeekable interface {
	IntIterable

	PeekNext() uint32
	AdvanceIfNeeded(minval uint32)
}

type Bitmap interface {

	// these methods manipulate the bitmap
	Add(uint32)
	AddMany([]uint32)
	AddRange(uint64,  uint64)
	And(Bitmap)
	AndNot(Bitmap)
	Clone() Bitmap
	Contains(uint32) bool
	GetCardinality() uint64
	GetSizeInBytes() uint64
	IsEmpty() bool
	Iterator() IntPeekable
	ReadFrom(io.Reader) (int64, error)
	WriteTo(io.Writer) (int64, error)

	// these methods return new bitmaps
	OrNew(Bitmap) Bitmap
	AndNew(Bitmap) Bitmap
	AndNotNew(Bitmap) Bitmap
	HeapOrNew(bitmaps ...Bitmap) Bitmap
}
