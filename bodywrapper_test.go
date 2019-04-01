package witness

import (
	"io"
	"testing"
)

func TestBodyWrapperRead(t *testing.T) {
	bw := &bodyWrapper{body: nil}
	var p []byte
	_, err := bw.Read(p)
	if err != io.EOF {
		t.Error("expected EOF")
	}
}

func TestBodyWrapperClose(t *testing.T) {
	bw := &bodyWrapper{body: nil, onClose: func(bw *bodyWrapper) {}}
	err := bw.Close()
	if err != nil {
		t.Error(err)
	}
}
