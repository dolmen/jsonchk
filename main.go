package main

import (
	"fmt"
	"github.com/xeipuuv/gojsonschema"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Context struct {
	schema  *gojsonschema.Schema
	result  chan<- string
	running chan<- bool
}

func validateFile(ctx Context, path string) {
	ctx.running <- true
	defer func() {
		ctx.running <- false
	}()

	//fmt.Println(path)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		ctx.result <- fmt.Sprint(err)
		return
	}
	result, err := ctx.schema.Validate(gojsonschema.NewStringLoader(string(bytes)))
	if err != nil {
		ctx.result <- fmt.Sprint(path, err)
		return
	}

	if !result.Valid() {
		errors := result.Errors()
		res := make([]string, 1+len(errors))
		res[0] = fmt.Sprintf("%s:", path)
		//ctx.result <- fmt.Sprintf("%s:\n", path)
		for i, desc := range errors {
			res[1+i] = fmt.Sprint(desc)
		}
		ctx.result <- strings.Join(res, "\n- ") + "\n"
	} else {
		//ctx.result <- = fmt.Sprintf("%s: ok\n", path)
	}
}

func validateDir(ctx Context, dir string) {
	ctx.running <- true
	defer func() {
		ctx.running <- false
	}()

	file, err := os.Open(dir)
	if err != nil {
		ctx.result <- fmt.Sprintln(err)
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
					go validateDir(ctx, path)
				} else {
					go validateFile(ctx, path)
				}
			}
		} else {
			if errDir != io.EOF {
				ctx.result <- fmt.Sprintln(err)
			}
			break
		}
	}
}

func validateAny(ctx Context, path string) {
	if fileinfo, err := os.Stat(path); err == nil {
		if fileinfo.IsDir() {
			go validateDir(ctx, path)
		} else {
			go validateFile(ctx, path)
		}
	} else {
		panic(err)
	}
}

func main() {
	result := make(chan string)
	running := make(chan bool)
	ctx := Context{
		result:  result,
		running: running,
	}

	schemaFile, err := filepath.Abs(os.Args[1])
	u := url.URL{
		Scheme: "file",
		Path:   schemaFile,
	}
	ctx.schema, err = gojsonschema.NewSchema(gojsonschema.NewReferenceLoader(u.String()))
	if err != nil {
		panic(err)
	}

	if len(os.Args) == 2 {
		fmt.Println("missing files/dir to process")
		return
	}

	for _, arg := range os.Args[2:] {
		validateAny(ctx, arg)
	}
	count := 0
	for {
		select {
		case r := <-running:
			if r {
				count++
			} else {
				count--
				if count == 0 {
					//fmt.Println("Done.")
					return
				}
			}
			//fmt.Println("Running: ", count)
		case r := <-result:
			fmt.Print(r)
		}
	}
}
