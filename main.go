package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gta_save/save"
	"io"
	"os"
	"path"
)

// This should be split up into a bunch of more flexible functions (or methods?) in the future.
// Currently this is just experimental.
func doEmbedding(input *os.File, output *os.File) {
	platform := save.NewGamePlatform(input)
	fmt.Printf("Detected platform: %s\n", platform.ToString())

	block0 := save.ReadVarBlock(&platform, input)
	scripts := save.ReadScriptBlock(&platform, input)

	const expandedByteCount = 60000

	// Calculate the space we're adding so we know how much we have to play with.
	oldSpace := scripts.GlobalByteCount()
	addedSpace := expandedByteCount - oldSpace

	fmt.Printf("Adding %d bytes to global store.\n", addedSpace)
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

	targetLength := 195000

	if platform.IsPC {
		targetLength = 202752
	}

	// TODO: Target length for PS2 platform.

	buffer := bytes.NewBuffer(make([]byte, 0, targetLength))

	save.WriteVarBlock(&platform, buffer, &block0)
	save.WriteScriptBlock(&platform, buffer, &scripts)

	// Copy the remaining data from the input file to the output buffer.
	readBuffer := make([]byte, 512)

	for {
		read, err := input.Read(readBuffer)

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

	length := buffer.Len()

	if platform.IsPC {
		length = targetLength
	}

	finalBytes := buffer.Bytes()[:length-4]

	var checksum uint32 = 0
	for _, value := range finalBytes {
		checksum += uint32(value)
	}

	checksumBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(checksumBytes, checksum)

	finalBytes = append(finalBytes, checksumBytes...)

	_, err := output.Write(finalBytes)

	if err != nil {
		panic(err)
	}
}

func main() {
	arguments := os.Args[1:]

	if len(arguments) != 2 {
		fileName := path.Base(os.Args[0])
		fmt.Printf("Usage: '%s <path to save> <destination for modded save>'\n", fileName)
		os.Exit(1)
	}

	inputFile, err := os.OpenFile(arguments[0], os.O_RDONLY, 0755)

	if err != nil {
		println("Error opening input file. Please check the path and try again.")
		os.Exit(1)
	}

	defer inputFile.Close()

	outputFile, err := os.OpenFile(arguments[1], os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)

	if err != nil {
		println("Error opening output file. Please check the path and that the destination is writeable.")
		os.Exit(1)
	}

	defer outputFile.Close()

	doEmbedding(inputFile, outputFile)
}
