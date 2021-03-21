package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gta_save/save"
	"io"
	"os"
	"path"

	"github.com/Squ1dd13/scm"
)

func translateOffsets(codeBytes []byte, byteOffset uint32) {
	reader := bytes.NewReader(codeBytes)

	getByteIndex := func() int64 {
		value, _ := reader.Seek(0, io.SeekCurrent)
		return value
	}

	// Disassemble each instruction so we can check if we need to patch them.
	for reader.Len() != 0 {
		instructionIndex := getByteIndex()
		instruction := scm.ReadInstruction(reader)

		if instruction == nil {
			println("Bad instruction, stopping.")
			break
		}

		const (
			jumpOpcode           = 0x0002
			falseJumpOpcode      = 0x004d
			callOpcode           = 0x0050
			switchOpcode         = 0x0871
			switchContinueOpcode = 0x0872
		)

		// TODO: Patch switches.

		switch instruction.Opcode {
		case jumpOpcode, falseJumpOpcode, callOpcode:
			{
				// Offset by 3 to skip past the opcode and type byte, then end at +7 after 4 bytes.
				addressSlice := codeBytes[instructionIndex+3 : instructionIndex+7]

				addressBuffer := bytes.NewBuffer(addressSlice)

				var address int32
				err := binary.Read(addressBuffer, binary.LittleEndian, &address)

				if err != nil {
					fmt.Printf("Failed to read address for jump/call at %d: %v\n", instructionIndex, err)
					continue
				}

				address = -address + int32(byteOffset)

				addressBuffer.Reset()
				binary.Write(addressBuffer, binary.LittleEndian, &address)

				println("Patched")
			}
		}
	}
}

// This should be split up into a bunch of more flexible functions (or methods?) in the future.
// Currently this is just experimental.
func doEmbedding(input *os.File, scriptBytes []byte, output *os.File) {
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

	// Translate jumps to match the embedded location.
	translateOffsets(scriptBytes, oldSpace)

	scripts.AddScript(&platform, &block0, "embed", scriptBytes, oldSpace)

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

	// TODO: Review this
	if platform.IsPC || (platform.IsMobile && length > targetLength) {
		println("Warning: Removing bytes from end of save to restrict length. " +
			"This will likely cause issues if these bytes are not padding.")

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

	if len(arguments) != 3 {
		fileName := path.Base(os.Args[0])
		fmt.Printf("Usage: '%s <path to save file> <path to script> <destination for modded save file>'\n", fileName)
		os.Exit(1)
	}

	inputFile, err := os.OpenFile(arguments[0], os.O_RDONLY, 0755)

	if err != nil {
		println("Error opening input file. Please check the path and try again.")
		os.Exit(1)
	}

	defer inputFile.Close()

	scriptBytes, err := os.ReadFile(arguments[1])

	if err != nil {
		println("Error opening script file. Please check the path and try again.")
		os.Exit(1)
	}

	outputFile, err := os.OpenFile(arguments[2], os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)

	if err != nil {
		println("Error opening output file. Please check the path and that the destination is writeable.")
		os.Exit(1)
	}

	defer outputFile.Close()

	doEmbedding(inputFile, scriptBytes, outputFile)
}
