package logging

import (
	"testing"
)

func TestNilForCoverage(t *testing.T) {
	logger := NewNilLogger()
	logger.Debugf("test")
	logger.Tracef("test")
	logger.Noticef("test")
	logger.Errorf("test")
	logger.Warnf("test")
	// skip fatal
	logger.Close()
}
