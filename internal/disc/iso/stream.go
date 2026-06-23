package iso

// StreamListener receives each sector of a file's raw user data in order, as the
// file is streamed. Returning a non-nil error stops the stream, and that error is
// returned by [FileStream.Stream].
type StreamListener interface {
	// Sector receives one sector's raw, form-aware user data and its XA
	// subheader, which is nil when the sector has none.
	Sector(data []byte, subheader *Subheader) error
}

// ListenerFunc adapts a function to a [StreamListener].
type ListenerFunc func(data []byte, subheader *Subheader) error

// Sector calls f.
func (f ListenerFunc) Sector(data []byte, subheader *Subheader) error {
	return f(data, subheader)
}

var _ StreamListener = ListenerFunc(nil)

// FileStream reads a file's raw, form-aware user data sector by sector. It is
// produced by the reader and handed to [Visitor.VisitFile]; its zero value
// streams nothing.
type FileStream struct {
	source SectorSource
	start  int
	end    int
	length int64
}

// Stream reads the file's sectors and delivers each to listener: the sector's
// logical block, trimmed on the final sector to the file's length, followed by
// any stream tail, along with the sector's XA subheader. It returns the first
// error from listener or from reading a sector.
func (s *FileStream) Stream(listener StreamListener) error {
	remaining := s.length
	for n := s.start; n < s.end; n++ {
		sector, err := s.source.ReadSector(n)
		if err != nil {
			return err
		}
		block := sector.Block
		if int64(len(block)) > remaining {
			block = block[:remaining]
		}
		remaining -= int64(len(block))
		data := block
		if sector.TailIsStream {
			data = append(append(make([]byte, 0, len(block)+len(sector.Tail)), block...), sector.Tail...)
		}
		if err := listener.Sector(data, sector.Subheader); err != nil {
			return err
		}
	}
	return nil
}
