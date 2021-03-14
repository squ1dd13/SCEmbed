package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gtasave/save"
	"io"
	"os"
)

func main() {
	file, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0755)
	defer file.Close()

	if err != nil {
		panic(err)
	}

	platform := save.NewGamePlatform(file)

	block0 := save.ReadVarBlock(&platform, file)
	scripts := save.ReadScriptBlock(&platform, file)

	// 60028 is the lower bound for expanded sizes on all platforms.
	const expandedByteCount uint32 = 60028

	// Calculate the space we're adding so we know how much we have to play with.
	oldSpace := scripts.GlobalStorage.GlobalSpaceSize
	addedSpace := expandedByteCount - oldSpace

	fmt.Printf("Adding %d bytes to global store.", addedSpace)
	scripts.ExpandGlobalSpace(int(expandedByteCount) / 4)

	scriptBytes := []byte{
		// wait 4000 ms
		0x01, 0x00, 0x05, 0xA0, 0x0F,

		// show_text_styled GXT 'FESZ_LS' time 4000 style 4
		0xBA, 0x00, 0x09, 0x46, 0x45, 0x53, 0x5A, 0x5F, 0x4C, 0x53, 0x00, 0x05, 0xA0, 0x0F, 0x04, 0x04,

		// play_music mission_complete
		0x94, 0x03, 0x04, 0x02,

		// jump (without destination; we have to add that)
		0x02, 0x00, 0x01 /* destination --> */, 0x0, 0x0, 0x0, 0x0,
	}

	binary.LittleEndian.PutUint32(scriptBytes[len(scriptBytes)-4:], scripts.ScriptAt(1).Info.RelativeInstructionPointer)

	// Very inefficient...
	for len(scriptBytes)%4 != 0 {
		scriptBytes = append(scriptBytes, 0)
	}

	scripts.ScriptAt(1).Info.RelativeInstructionPointer = oldSpace

	for i := 0; i < len(scriptBytes); i += 4 {
		globalValue := binary.LittleEndian.Uint32(scriptBytes[i : i+4])

		globalIndex := (int(oldSpace) + i) / 4
		scripts.GlobalStorage.Globals[globalIndex] = globalValue
	}

	buffer := bytes.NewBuffer(make([]byte, 0, 195_000))

	save.WriteVarBlock(&platform, buffer, &block0)
	save.WriteScriptBlock(&platform, buffer, &scripts)

	// Copy the remaining data from the input file to the output buffer.
	readBuffer := make([]byte, 512)

	for {
		read, err := file.Read(readBuffer)

		if read == 0 {
			break
		}

		if err != nil && err != io.EOF {
			panic(err)
		}

		_, err = buffer.Write(readBuffer[:read])

		if err != nil {
			panic(err)
		}
	}

	finalBytes := buffer.Bytes()[:buffer.Len()-4]

	var checksum uint32 = 0
	for _, value := range finalBytes {
		checksum += uint32(value)
	}

	checksumBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(checksumBytes, checksum)

	finalBytes = append(finalBytes, checksumBytes...)

	outFile, err := os.OpenFile(
		os.Args[1]+".modded",
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0755)

	defer outFile.Close()

	if err != nil {
		panic(err)
	}

	_, err = outFile.Write(finalBytes)

	if err != nil {
		panic(err)
	}
}
