package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ishanjain28/plato"
)

func main() {
	f, err := plato.Download("https://www.ishanjain.me/projects/projects", 10)
	if err != nil {
		log.Fatalln(err.Error())
	}
	file, err := os.Create("test")
	if err != nil {
		log.Fatalln(err.Error())
	}

	b, _ := ioutil.ReadAll(f)

	fmt.Println(b)
	defer file.Close()
	defer f.Close()

}
