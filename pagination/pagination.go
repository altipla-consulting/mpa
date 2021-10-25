package pagination

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"

	"libs.altipla.consulting/errors"
	"libs.altipla.consulting/rdb"
)

const (
	DefaultPageSize = 30
	MaxPageSize     = 1000
)

type Info struct {
	loadErr        error
	page, pageSize int64
	start, end     int64
	q              *rdb.Query
	totalSize      int64
}

func New(r *http.Request, q *rdb.Query) *Info {
	info := &Info{
		pageSize: DefaultPageSize,
		q:        q,
		page:     1,
	}

	if pageSize := r.FormValue("page-size"); pageSize != "" {
		n, err := strconv.ParseInt(pageSize, 10, 64)
		if err == nil {
			info.pageSize = n
		}
		if n > MaxPageSize {
			info.pageSize = MaxPageSize
		}
	}

	if page := r.FormValue("page"); page != "" {
		n, err := strconv.ParseInt(page, 10, 64)
		if err == nil {
			info.page = n
		}
		if info.page < 1 {
			info.page = 1
		}
	}

	info.start = (info.page - 1) * info.pageSize

	return info
}

func (info *Info) Fetch(ctx context.Context, dest interface{}, opts ...rdb.IncludeOption) error {
	q := info.q.Limit(info.pageSize).Offset(info.start)
	if err := q.GetAll(ctx, dest, opts...); err != nil {
		return errors.Trace(err)
	}

	info.end = info.start + int64(reflect.ValueOf(dest).Elem().Len())
	info.totalSize = info.q.Stats().TotalResults

	return nil
}

type repr struct {
	Next        int64 `json:"next"`
	Prev        int64 `json:"prev"`
	Start       int64 `json:"start"`
	End         int64 `json:"end"`
	TotalSize   int64 `json:"totalSize"`
	OutOfBounds bool  `json:"outOfBounds"`
}

func (info *Info) Info() (string, error) {
	r := repr{
		Start:     info.start + 1,
		End:       info.end,
		TotalSize: info.totalSize,
	}
	if info.page > 1 {
		r.Prev = info.page - 1
	}
	if info.totalSize > info.end {
		r.Next = info.page + 1
	}
	if info.start >= info.end {
		r.OutOfBounds = true

		r.Prev = info.q.Stats().TotalResults / info.pageSize
		if info.q.Stats().TotalResults%info.pageSize != 0 {
			r.Prev++
		}
	}

	b, err := json.Marshal(r)
	if err != nil {
		return "", errors.Trace(err)
	}
	return string(b), nil
}

func (info *Info) TotalSize() int64 {
	return info.totalSize
}
