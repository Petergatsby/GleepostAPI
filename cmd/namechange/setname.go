package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
)

func main() {
	flag.Parse()
	path := flag.Arg(0)
	pbxproj, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	re := regexp.MustCompile("PRODUCT_NAME = \\w+;")
	replaced := re.ReplaceAll(pbxproj, []byte(fmt.Sprintf("PRODUCT_NAME = %s;", flag.Arg(1))))
	err = ioutil.WriteFile(path, replaced, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
