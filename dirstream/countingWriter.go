package dirstream

import "io"

// CountingWriter wraps an io.Writer and counts the number of bytes written.
type CountingWriter struct {
	w     io.Writer
	Count uint64
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.Count += uint64(n)
	return n, err
}
