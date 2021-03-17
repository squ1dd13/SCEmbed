package save

import (
	"encoding/binary"
	"io"
	"os"
	"strings"
)

// Identifies the type of a save file, and provides version-dependent values.
type GamePlatform struct {
	IsMobile   bool
	IsWideChar bool
	IsPS2      bool
	IsPC       bool
}

func NewGamePlatform(file *os.File) GamePlatform {
	// Make sure that we don't change the file position from the beginning.
	defer file.Seek(0, 0)

	fileInfo, err := file.Stat()

	if err != nil {
		panic(err)
	}

	fileSize := fileInfo.Size()

	platform := GamePlatform{}

	blockBytes := make([]byte, 5)
	_, err = file.ReadAt(blockBytes, 333)

	if err != nil {
		panic(err)
	}

	intBytes := make([]byte, 4)
	_, err = file.ReadAt(intBytes, 46516)

	if err != nil {
		panic(err)
	}

	ps2Int := binary.LittleEndian.Uint32(intBytes)

	isPs2Japan := string(blockBytes) == "BLOCK"

	platform.IsMobile = fileSize == 195_000 || fileSize == 260_000 || fileSize == 325_000 || fileSize == 390_000
	platform.IsPS2 = isPs2Japan || ps2Int == 0x2fc86
	platform.IsWideChar = isPs2Japan || platform.IsMobile

	platform.IsPC = !platform.IsMobile && !platform.IsPS2

	return platform
}

func (platform *GamePlatform) MaxLocals() int {
	if platform.IsMobile {
		return 40
	}

	return 32
}

func (platform *GamePlatform) ToString() string {
	if platform.IsMobile {
		return "Mobile"
	}

	if platform.IsPC {
		return "PC"
	}

	if platform.IsPS2 {
		return "PS2"
	}

	return "Unknown"
}

// Makes it clearer that a field is just padding.
type padding uint8

// Booleans are single byte values in saves, and since save files
//  are dumps from C++ structures, they are padded, and thus bools
//  are often followed by three bytes of padding to make the size
//  aligned to a four-byte boundary.
type boolPadding [3]uint8

type vector3 struct {
	X, Y, Z float32
}

func mustRead(file io.Reader, data interface{}) {
	err := binary.Read(file, binary.LittleEndian, data)

	if err != nil {
		panic(err)
	}
}

func mustWrite(file io.Writer, data interface{}) {
	err := binary.Write(file, binary.LittleEndian, data)

	if err != nil {
		panic(err)
	}
}

func nullTerminate(str *string) {
	index := strings.IndexRune(*str, '\x00')
	*str = (*str)[:index]
}
