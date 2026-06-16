package disc

// Option configures a [Disc] when passed to [NewDisc].
type Option interface {
	apply(*Disc)
}

// option adapts a plain function to the [Option] interface.
type option func(*Disc)

func (o option) apply(d *Disc) { o(d) }

// WithOpenFunc sets the [OpenFunc] used to open the disc's backing files. The
// default opens files from the local filesystem.
func WithOpenFunc(open OpenFunc) Option {
	return option(func(d *Disc) {
		d.open = open
	})
}

// WithStatFunc sets the [StatFunc] used to size the disc's backing files. The
// default stats files on the local filesystem.
func WithStatFunc(stat StatFunc) Option {
	return option(func(d *Disc) {
		d.stat = stat
	})
}

// WithSectorCallback registers cb to receive a report for every sector whose
// stored EDC does not match its computed value. When set, such sectors are
// decoded best-effort instead of aborting a read. When unset, a checksum
// mismatch surfaces as an error from the track's reads.
func WithSectorCallback(cb func(error)) Option {
	return option(func(d *Disc) {
		d.sectorCallback = cb
	})
}
