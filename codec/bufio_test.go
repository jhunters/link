package codec

import (
	"encoding/binary"
	"testing"
)

func Test_Bufio(t *testing.T) {
	codecTest(t, Bufio[MyMessage1, MyMessage1](FixLen[MyMessage1, MyMessage1](JsonTestProtocol(), 2, binary.LittleEndian, 64*1024, 64*1024), 1024, 1024))
}
