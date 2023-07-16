package buffer

import (
	"io"
)

type RepeatableBuffer struct {
	Buffer
}

func NewRepeatableBuffer() *RepeatableBuffer {
	return &RepeatableBuffer{}
}

func (rb *RepeatableBuffer) Fork() *RepeatableBufferFork {
	return NewRepeatableBufferFork(&rb.Buffer, 0)
}

type RepeatableBufferFork struct {
	origin *Buffer
	off    int
}

func NewRepeatableBufferFork(buf *Buffer, off int) *RepeatableBufferFork {
	return &RepeatableBufferFork{
		origin: buf,
		off:    off,
	}
}

func (rbf *RepeatableBufferFork) empty() bool {
	return len(rbf.origin.buf) <= rbf.off
}

func (rbf *RepeatableBufferFork) Read(p []byte) (n int, err error) {
	if rbf.empty() {
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	n = copy(p, rbf.origin.buf[rbf.off:])
	rbf.off += n
	return n, nil
}

func (rbf *RepeatableBufferFork) Bytes() []byte {
	return rbf.origin.buf[rbf.off:]
}

func (rbf *RepeatableBufferFork) String() string {
	return string(rbf.Bytes())
}

func (rbf *RepeatableBufferFork) Fork() *RepeatableBufferFork {
	return NewRepeatableBufferFork(rbf.origin, 0)
}
