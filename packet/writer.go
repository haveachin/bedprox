package packet

import (
	"io"
	"unsafe"
)

type Writer struct {
	w interface {
		io.Writer
		io.ByteWriter
	}
}

func NewWriter(w interface {
	io.Writer
	io.ByteWriter
}) *Writer {
	return &Writer{w: w}
}

func (w *Writer) Bool(x *bool) {
	_ = w.w.WriteByte(*(*byte)(unsafe.Pointer(x)))
}

func (w *Writer) String(x *string) {
	l := uint32(len(*x))
	w.Varuint32(&l)
	_, _ = w.w.Write([]byte(*x))
}

func (w *Writer) Varuint32(x *uint32) {
	u := *x
	for u >= 0x80 {
		_ = w.w.WriteByte(byte(u) | 0x80)
		u >>= 7
	}
	_ = w.w.WriteByte(byte(u))
}
