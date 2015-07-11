package byteutils

func Cut(a []byte, from, to int) []byte {
	copy(a[from:], a[to:])
	a = a[:len(a)-to+from]

	return a
}

func Insert(a []byte, i int, b []byte) []byte {
	a = append(a, make([]byte, len(b))...)
	copy(a[i+len(b):], a[i:])
	copy(a[i:i+len(b)], b)

	return a
}

// Unlike bytes.Replace it allows you to specify range
func Replace(a []byte, from, to int, new []byte) []byte {
	lenDiff := len(new) - (to - from)

	if lenDiff > 0 {
		// Extend if new segment bigger
		a = append(a, make([]byte, lenDiff)...)
		copy(a[to+lenDiff:], a[to:])
		copy(a[from:from+len(new)], new)

		return a
	} else if lenDiff < 0 {
		copy(a[from:], new)
		copy(a[from+len(new):], a[to:])
		return a[:len(a)+lenDiff]
	} else { // same size
		copy(a[from:], new)
		return a
	}
}
