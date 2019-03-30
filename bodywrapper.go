package witness

import (
	"io"
	"time"
)

type BodyWrapper struct {
	body             io.ReadCloser
	readingStartedAt time.Time
	readingStoppedAt time.Time
	content          []byte
	onClose          func(*BodyWrapper)
}

func (bw *BodyWrapper) Read(p []byte) (n int, err error) {
	if bw.readingStartedAt.IsZero() {
		bw.readingStartedAt = time.Now()
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
		bw.readingStoppedAt = time.Now()
		return
	}
	return
}

func (bw *BodyWrapper) Close() (err error) {
	bw.onClose(bw)
	if bw.body != nil {
		return bw.body.Close()
	}
	return nil
}
