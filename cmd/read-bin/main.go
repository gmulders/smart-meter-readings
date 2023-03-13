package main

import (
	//	"encoding/binary"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
)

func main() {

	//	binary.ReadVarint()
	content, err := ioutil.ReadFile("meterstanden-2023-01.002.bin")
	if err != nil {
		log.Fatal(err)
	}

	reader := bytes.NewReader(content)

	var i int
	for reader.Len() > 0 {
		value, err := binary.ReadVarint(reader)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%d: %d\n", i, value)
		i++
	}
}

//	n := binary.PutVarint(buff, newValue-oldValue)
