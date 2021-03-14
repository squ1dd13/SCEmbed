package main

import (
	"bytes"
	"encoding/binary"
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

	block0 := save.ReadVarBlock(file)
	scripts := save.ReadScriptBlock(file)

	buffer := bytes.NewBuffer(make([]byte, 0, 195_000))

	save.WriteVarBlock(buffer, &block0)
	save.WriteScriptBlock(buffer, &scripts)

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
