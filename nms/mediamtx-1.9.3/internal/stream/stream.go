// Package stream contains the Stream object.
package stream

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"

	"github.com/bluenviron/mediamtx/internal/logger"
	"github.com/bluenviron/mediamtx/internal/unit"
)

// Reader is a stream reader.
type Reader interface {
	logger.Writer
}

// ReadFunc is the callback passed to AddReader().
type ReadFunc func(unit.Unit) error

// Stream is a media stream.
// It stores tracks, readers and allows to write data to readers.
type Stream struct {
	writeQueueSize int
	desc           *description.Session

	bytesReceived *uint64
	bytesSent     *uint64
	streamMedias  map[*description.Media]*streamMedia
	mutex         sync.RWMutex
	rtspStream    *gortsplib.ServerStream
	rtspsStream   *gortsplib.ServerStream
	streamReaders map[Reader]*streamReader

	readerRunning chan struct{}
}

// New allocates a Stream.
func New(
	writeQueueSize int,
	udpMaxPayloadSize int,
	desc *description.Session,
	generateRTPPackets bool,
	decodeErrLogger logger.Writer,
) (*Stream, error) {
	s := &Stream{
		writeQueueSize: writeQueueSize,
		desc:           desc,
		bytesReceived:  new(uint64),
		bytesSent:      new(uint64),
	}

	s.streamMedias = make(map[*description.Media]*streamMedia)
	s.streamReaders = make(map[Reader]*streamReader)
	s.readerRunning = make(chan struct{})

	for _, media := range desc.Medias {
		var err error
		s.streamMedias[media], err = newStreamMedia(udpMaxPayloadSize, media, generateRTPPackets, decodeErrLogger)
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

// Close closes all resources of the stream.
func (s *Stream) Close() {
	if s.rtspStream != nil {
		s.rtspStream.Close()
	}
	if s.rtspsStream != nil {
		s.rtspsStream.Close()
	}
}

// Desc returns the description of the stream.
func (s *Stream) Desc() *description.Session {
	return s.desc
}

// BytesReceived returns received bytes.
func (s *Stream) BytesReceived() uint64 {
	return atomic.LoadUint64(s.bytesReceived)
}

// BytesSent returns sent bytes.
func (s *Stream) BytesSent() uint64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	bytesSent := atomic.LoadUint64(s.bytesSent)
	if s.rtspStream != nil {
		bytesSent += s.rtspStream.BytesSent()
	}
	if s.rtspsStream != nil {
		bytesSent += s.rtspsStream.BytesSent()
	}
	return bytesSent
}

// RTSPStream returns the RTSP stream.
func (s *Stream) RTSPStream(server *gortsplib.Server) *gortsplib.ServerStream {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.rtspStream == nil {
		s.rtspStream = gortsplib.NewServerStream(server, s.desc)
	}
	return s.rtspStream
}

// RTSPSStream returns the RTSPS stream.
func (s *Stream) RTSPSStream(server *gortsplib.Server) *gortsplib.ServerStream {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.rtspsStream == nil {
		s.rtspsStream = gortsplib.NewServerStream(server, s.desc)
	}
	return s.rtspsStream
}

// AddReader adds a reader.
// Used by all protocols except RTSP.
func (s *Stream) AddReader(reader Reader, medi *description.Media, forma format.Format, cb ReadFunc) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	sr, ok := s.streamReaders[reader]
	if !ok {
		sr = &streamReader{
			queueSize: s.writeQueueSize,
			parent:    reader,
		}
		sr.initialize()

		s.streamReaders[reader] = sr
	}

	sm := s.streamMedias[medi]
	sf := sm.formats[forma]
	sf.addReader(sr, cb)
}

// RemoveReader removes a reader.
// Used by all protocols except RTSP.
func (s *Stream) RemoveReader(reader Reader) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	sr := s.streamReaders[reader]

	for _, sm := range s.streamMedias {
		for _, sf := range sm.formats {
			sf.removeReader(sr)
		}
	}

	delete(s.streamReaders, reader)

	sr.stop()
}

// StartReader starts a reader.
// Used by all protocols except RTSP.
func (s *Stream) StartReader(reader Reader) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	sr := s.streamReaders[reader]

	sr.start()

	for _, sm := range s.streamMedias {
		for _, sf := range sm.formats {
			sf.startReader(sr)
		}
	}

	select {
	case <-s.readerRunning:
	default:
		close(s.readerRunning)
	}
}

// ReaderError returns whenever there's an error.
func (s *Stream) ReaderError(reader Reader) chan error {
	sr := s.streamReaders[reader]
	return sr.error()
}

// ReaderFormats returns all formats that a reader is reading.
func (s *Stream) ReaderFormats(reader Reader) []format.Format {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sr := s.streamReaders[reader]
	var formats []format.Format

	for _, sm := range s.streamMedias {
		for forma, sf := range sm.formats {
			if _, ok := sf.pausedReaders[sr]; ok {
				formats = append(formats, forma)
			} else if _, ok := sf.runningReaders[sr]; ok {
				formats = append(formats, forma)
			}
		}
	}

	return formats
}

// WaitRunningReader waits for a running reader.
func (s *Stream) WaitRunningReader() {
	<-s.readerRunning
}

// WriteUnit writes a Unit.
func (s *Stream) WriteUnit(medi *description.Media, forma format.Format, u unit.Unit) {
	sm := s.streamMedias[medi]
	sf := sm.formats[forma]

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sf.writeUnit(s, medi, u)
}

// WriteRTPPacket writes a RTP packet.
func (s *Stream) WriteRTPPacket(
	medi *description.Media,
	forma format.Format,
	pkt *rtp.Packet,
	ntp time.Time,
	pts int64,
) {
	sm := s.streamMedias[medi]
	sf := sm.formats[forma]

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sf.writeRTPPacket(s, medi, pkt, ntp, pts)
}