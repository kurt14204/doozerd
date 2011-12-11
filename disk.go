package logfs

// On disk format for logfs.

// A block is the fundamental unit on disk.  Each Record translates
// to a block on disk.
type block struct {
	header blockHeader // Records information about data.
	data   []byte      // Passed by Write, returned to Read.
	next   uint64      // File offset of the next block.
}

// blockHeader describes a block.  A block is variable sized, so this
// header is fixed size to know how much to read.
type blockHeader struct {
	headerSum  uint64 // Own checksum.
	cookie     uint64 // To be matched with previous block.
	nextCookie uint64 // Next block's cookie must match this.
	dataLen    uint64 // block.data length.
	dataSum    uint64 // block.data checksum.
	seqn       uint64 // Sequence number, must grow in the list.
}

// physWrite issues a write to the writer, waits for the result
// and returns it.  It multiplexes many clients to one writer.
func (l *Logfs) physWrite(b *block) error {
	c := make(chan error)
	l.w <- iop{b, c}
	return <-c
}

// physRead issues a read request to the reader, waits for
// the result and returns it.  It multiplexes many clients
// to one reader.
func (l *Logfs) physRead() (b *block, err error) {
	c := make(chan error)
	l.r <- iop{b, c}
	return b, <-c
}
