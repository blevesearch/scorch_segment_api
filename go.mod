module github.com/blevesearch/scorch_segment_api/v2

go 1.19

require (
	github.com/RoaringBitmap/roaring v1.2.3
	github.com/blevesearch/bleve_index_api v1.0.6
)

require (
	github.com/bits-and-blooms/bitset v1.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
)

replace github.com/blevesearch/bleve_index_api => ../bleve_index_api
