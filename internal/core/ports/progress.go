package ports

import (
	"context"
	"io"
)

// Progress is optional display telemetry. It is never execution, Evidence or
// Policy authority. Unknown totals are zero; bytes received do not mean verified.
type Progress struct {
	Stage   string
	Current int64
	Total   int64
	Unit    string
}

type progressContextKey struct{}

// WithProgress attaches a caller-owned, synchronous, non-authorizing observer.
// Observers must be bounded and must not panic or perform workflow operations.
func WithProgress(ctx context.Context, observe func(Progress)) context.Context {
	return context.WithValue(ctx, progressContextKey{}, observe)
}

// ReportProgress reports actual work without changing any operation result.
func ReportProgress(ctx context.Context, event Progress) {
	if ctx == nil {
		return
	}
	if observe, ok := ctx.Value(progressContextKey{}).(func(Progress)); ok && observe != nil {
		observe(event)
	}
}

// DownloadReader counts bytes actually read inside the existing bounded reader.
// It adds no buffering, retries, authority or changes to read errors/budgets.
func DownloadReader(ctx context.Context, reader io.Reader, total int64) io.Reader {
	return &downloadProgressReader{ctx: ctx, reader: reader, total: total}
}

type downloadProgressReader struct {
	ctx            context.Context
	reader         io.Reader
	total, current int64
}

func (r *downloadProgressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.current += int64(n)
	if n > 0 {
		ReportProgress(r.ctx, Progress{Stage: "download", Current: r.current, Total: r.total, Unit: "bytes"})
	}
	return n, err
}
