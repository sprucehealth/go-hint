package hint

import (
	"net/http"
	"strconv"
)

// ListMeta is the structure that contains the common properties
// of List iterators.
type ListMeta struct {
	CurrentCount uint64 `json:"count"`
	TotalCount   uint64 `json:"total_count"`
}

// parseListMeta reads the pagination counts Hint returns on list endpoints from
// the response headers. Absent headers leave the corresponding count at zero.
func parseListMeta(resHeaders http.Header) (ListMeta, error) {
	var meta ListMeta
	var err error

	if xCountHeader := resHeaders.Get("x-count"); xCountHeader != "" {
		meta.CurrentCount, err = strconv.ParseUint(xCountHeader, 10, 64)
		if err != nil {
			return meta, err
		}
	}

	if xTotalCountHeader := resHeaders.Get("x-total-count"); xTotalCountHeader != "" {
		meta.TotalCount, err = strconv.ParseUint(xTotalCountHeader, 10, 64)
		if err != nil {
			return meta, err
		}
	}

	return meta, nil
}

// Query is the function used to get a page listing.
type Query func(params *ListParams) ([]interface{}, ListMeta, error)

// Iter is a structure used for generic pagination through a list of resources.
// NextPage is retrieved by calling the Query function.
type Iter struct {
	query        Query
	err          error
	cur          interface{}
	values       []interface{}
	totalQueried uint64
	hasMore      bool
	params       *ListParams
	meta         ListMeta
	// startOffset is the offset the caller asked to start paginating from, so
	// subsequent pages resume after it rather than from the start of the list.
	startOffset uint64
}

// GetIter returns an implementation of an iterator based on the
// params and the query function.
func GetIter(params *ListParams, query Query) *Iter {
	if params == nil {
		params = &ListParams{}
	}
	iter := &Iter{
		params:      params,
		startOffset: params.Offset,
	}
	iter.query = query
	iter.getPage()
	return iter
}

func (it *Iter) getPage() error {
	it.values, it.meta, it.err = it.query(it.params)
	it.totalQueried += uint64(len(it.values))
	// TotalCount counts every record matching the filters, so the starting offset
	// counts towards it as well.
	it.hasMore = it.startOffset+it.totalQueried < it.meta.TotalCount
	return it.err
}

// Next returns the next item in the list of resources, querying the source
// for the next page if there is more to query. It returns false when done querying
// or the query results in an error.
func (it *Iter) Next() bool {
	if len(it.values) == 0 && it.hasMore {
		it.params.Offset = it.startOffset + it.totalQueried
		if err := it.getPage(); err != nil {
			return false
		}
	}
	if len(it.values) == 0 {
		return false
	}

	it.cur = it.values[0]
	it.values = it.values[1:]
	return true
}

// Current returns the current item the iterator points to.
func (it *Iter) Current() interface{} {
	return it.cur
}

// Err returns an error the iterator is holding on to as a result
// of querying against list of resources.
func (it *Iter) Err() error {
	return it.err
}

// Count returns the total number of elements in the resulting interable
func (it *Iter) Count() uint64 {
	return it.meta.TotalCount
}

// Offset returns the offset from which the last page was received
func (it *Iter) Offset() uint64 {
	return it.params.Offset
}

// Offset returns true if there is at least another page of results to query for\
func (it *Iter) HasMore() bool {
	return it.hasMore
}
