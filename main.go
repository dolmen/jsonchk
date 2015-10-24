package main

import (
	"fmt"
	"github.com/xeipuuv/gojsonschema"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func validateFile(c chan<- string, schema *gojsonschema.Schema, path string) {
	//fmt.Println(path)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		c <- fmt.Sprint(err)
		return
	}
	result, err := schema.Validate(gojsonschema.NewStringLoader(string(bytes)))
	if err != nil {
		c <- fmt.Sprint(path, err)
		return
	}

	if !result.Valid() {
		errors := result.Errors()
		res := make([]string, 1+len(errors))
		res[0] = fmt.Sprintf("%s:", path)
		//c <- fmt.Sprintf("%s:\n", path)
		for i, desc := range errors {
			res[1+i] = fmt.Sprint(desc)
			//c <- fmt.Sprintln("- ", desc)
		}
		c <- strings.Join(res, "\n- ") + "\n"
	} else {
		//c <- fmt.Sprintf("%s: ok\n", path)
	}
}

func validateDir(c chan<- string, schema *gojsonschema.Schema, dir string) {
	file, err := os.Open(dir)
	if err != nil {
		c <- fmt.Sprintln(err)
		return
	}
	defer file.Close()

	for {
		fi, errDir := file.Readdir(50)
		if errDir == nil {
			for _, any := range fi {
				path := dir + "/" + any.Name()
				if any.IsDir() {
					// Recurse
					go validateDir(c, schema, path)
				} else {
					go validateFile(c, schema, path)
				}
			}
		} else {
			if errDir != io.EOF {
				c <- fmt.Sprintln(err)
			}
			break
		}
	}
}

func validateAny(c chan<- string, schema *gojsonschema.Schema, path string) {
	if fileinfo, err := os.Stat(path); err == nil {
		if fileinfo.IsDir() {
			go validateDir(c, schema, path)
		} else {
			//fmt.Println(path)
			go validateFile(c, schema, path)
		}
	} else {
		panic(err)
	}
}

func main() {
	c := make(chan string)
	schemaFile, err := filepath.Abs(os.Args[1])
	schema, err := gojsonschema.NewSchema(gojsonschema.NewReferenceLoader("file://" + schemaFile))
	if err != nil {
		panic(err)
	}

	for _, arg := range os.Args[2:] {
		validateAny(c, schema, arg)
	}
	for {
		fmt.Print(<-c)
	}
}
