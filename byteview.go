package geecache

type ByteView struct {
	b []byte
}

func (v ByteView) Len() uint {
	return uint(len(v.b))
}

func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
