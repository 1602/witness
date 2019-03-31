package witness

import (
	"io"
)

// BodyWrapper implements ReadCloser interface to wrap a body to spy on events.
type BodyWrapper struct {
	body           io.ReadCloser
	readingStarted bool
	content        []byte
	onReadingStart func()
	onReadingDone  func()
	onClose        func(*BodyWrapper)
}

// Read performs real read operation tracking time until completion.
func (bw *BodyWrapper) Read(p []byte) (n int, err error) {
	if !bw.readingStarted {
		bw.readingStarted = true
		if bw.onReadingStart != nil {
			bw.onReadingStart()
		}
	}
	if bw.body == nil {
		return 0, io.EOF
	}
	n, err = bw.body.Read(p)
	// fmt.Println(string(p), n, err)
	if bw.content == nil {
		bw.content = p[:n]
	} else {
		bw.content = append(bw.content, p[:n]...)
	}
	if err == io.EOF {
		// fmt.Println("Read body", now.Sub(bw.readingStartedAt))
		if bw.onReadingDone != nil {
			bw.onReadingDone()
		}
		return
	}
	return
}

// Close calls real body.Close and invokes internal callback to track time to closing.
func (bw *BodyWrapper) Close() (err error) {
	bw.onClose(bw)
	if bw.body != nil {
		return bw.body.Close()
	}
	return nil
}
