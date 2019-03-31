package witness

import (
	"io"
	"testing"
)

func TestBodyWrapperRead(t *testing.T) {
	bw := &BodyWrapper{body: nil}
	var p []byte
	_, err := bw.Read(p)
	if err != io.EOF {
		t.Error("expected EOF")
	}
}

func TestBodyWrapperClose(t *testing.T) {
	bw := &BodyWrapper{body: nil, onClose: func(bw *BodyWrapper) {}}
	err := bw.Close()
	if err != nil {
		t.Error(err)
	}
}
