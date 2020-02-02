package crypto

// ZeroReader is too public
type ZeroReader struct{}

func (ZeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

// or
// crypto/rand:  rand.Read(b []byte) (n int, err error)
