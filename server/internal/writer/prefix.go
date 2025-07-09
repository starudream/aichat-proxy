package writer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/starudream/aichat-proxy/server/internal/conv"
)

type PrefixWriter struct {
	w   io.Writer
	pre []byte
	buf bytes.Buffer
	mu  sync.Mutex
}

func NewPrefixWriter(pre string, ws ...io.Writer) *PrefixWriter {
	if len(ws) == 0 || ws[0] == nil {
		ws = []io.Writer{os.Stdout}
	}
	return &PrefixWriter{w: ws[0], pre: conv.StringToBytes(pre)}
}

var _ io.WriteCloser = (*PrefixWriter)(nil)

func (w *PrefixWriter) Write(bs []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(bs)
	for {
		idx := bytes.IndexByte(w.buf.Bytes(), '\n')
		if idx == -1 {
			break
		}
		_, _ = w.w.Write(w.concat(w.buf.Next(idx + 1)))
	}
	return len(bs), nil
}

func (w *PrefixWriter) concat(line []byte) []byte {
	return bytes.Join([][]byte{conv.StringToBytes(time.Now().Format("2006-01-02T15:04:05.000Z07:00")), w.pre, line}, []byte{' ', '|', ' '})
}

func (w *PrefixWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf.Len() > 0 {
		_, _ = w.w.Write(w.concat(w.buf.Bytes()))
		w.buf.Reset()
	}
	if c, ok := w.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (w *PrefixWriter) Printf(f string, a ...any) {
	_, _ = w.Write(conv.StringToBytes(fmt.Sprintf(f, a...)))
}
