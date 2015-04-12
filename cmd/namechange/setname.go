package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/draaglom/xctools/xcassets"
)

func main() {
	flag.Parse()
	path := flag.Arg(0)
	found, err := xcassets.FindPath(path, "project.pbxproj")
	if err != nil || len(found) == 0 {
		fmt.Println("Couldn't find project.pbxproj in this tree:", dest)
		os.Exit(-1)
	}
	if len(found) > 1 {
		fmt.Println("Destination ambiguous: Found multiple project.pbxproj in this tree:")
		for _, f := range found {
			fmt.Println(f)
		}
		os.Exit(-1)
	}
	pbxproj, err := ioutil.ReadFile(found[0])
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
