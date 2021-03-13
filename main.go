package main

import (
	"gtasave/save"
	"os"
)

func main() {
	file, err := os.OpenFile(os.Args[1], os.O_RDONLY, 0755)
	defer file.Close()

	if err != nil {
		panic(err)
	}

	save.ReadVarBlock(file)
	scripts := save.ReadScriptBlock(file)
	print(len(scripts.Brains))
	println("Hello, world!")
}
