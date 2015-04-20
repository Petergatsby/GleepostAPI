package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"image/color"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/draaglom/xctools/xcassets"
)

var names = []string{
	"firstAutoColour",
	"secondAutoColour",
	"thirdAutoColour",
	"fourthAutoColour",
	"fifthAutoColour",
	"sixthAutoColour",
}

func main() {
	//Take user input
	var primary = flag.String("primary", "", "The app's primary colour")
	var leftNav = flag.String("leftnav", "", "The left nav-bar colour")
	var rightNav = flag.String("rightnav", "", "The right nav-bar colour")
	var navBackground = flag.String("navbar", "", "The nav-bar background colour")
	var wallTitle = flag.String("walltitle", "", "The campus wall title colour")
	var navBarTitle = flag.String("navtitle", "", "The general title colour")
	colourFlags := []*string{primary, leftNav, rightNav, navBackground, wallTitle, navBarTitle}
	flag.Parse()
	args := flag.Args()
	filename := ""
	if len(args) < 1 {
		fmt.Println("You must supply the path to the GleepostIOS project.")
		fmt.Println("eg: colourchange -leftnav=fffeee ~/GleepostIOS")
		os.Exit(-1)
	}
	filename = args[0]
	colours := make(map[string]color.RGBA)
	for i, colour := range colourFlags {
		if *colour != "" {
			c, err := fromHex(*colour)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
			colours[names[i]] = c
		}
	}
	switch {
	case len(colours) == 0 && len(args) < 2:
		fmt.Println("Not enough colours supplied.")
		os.Exit(-1)
	case len(colours) == 0 && len(args) >= 2:
		for i, arg := range args[1:] {
			var name string
			c, err := fromHex(arg)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
			name = names[i]
			colours[name] = c
		}
	}
	found, err := xcassets.FindPath(filename, "AppearanceHelper.m")
	if err != nil || len(found) == 0 {
		fmt.Println("Couldn't find AppearanceHelper.m in this tree:", filename)
		os.Exit(-1)
	}
	if len(found) > 1 {
		fmt.Println("Destination ambiguous: Found multiple AppearanceHelper.m in this tree:")
		for _, f := range found {
			fmt.Println(f)
		}
		os.Exit(-1)
	}
	AppearanceHelper, err := ioutil.ReadFile(found[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	for name, col := range colours {
		AppearanceHelper, err = setColour(AppearanceHelper, name, col)
	}
	err = ioutil.WriteFile(found[0], AppearanceHelper, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func setColour(file []byte, colourName string, colour color.RGBA) ([]byte, error) {
	re, err := regexp.Compile(fmt.Sprintf("\\+ \\(UIColor \\*\\)%s\n{\n    return \\[UIColor colorWithR:\\d+ withG:\\d+ andB:\\d+\\];\n}", colourName))
	if err != nil {
		return file, err
	}
	replaced := re.ReplaceAll(file, []byte(colourM(colour, colourName)))
	return replaced, nil
}

func colourM(c color.RGBA, name string) string {
	return fmt.Sprintf("+ (UIColor *)%s\n{\n    return [UIColor colorWithR:%d withG:%d andB:%d];\n}", name, c.R, c.G, c.B)
}

func fromHex(col string) (c color.RGBA, err error) {
	if len(col) < 6 {
		return c, errors.New("Invalid colour")
	}
	if col[:1] == "#" {
		col = col[1:]
	}
	if len(col) > 6 {
		col = col[:6]
	}
	bytes, err := hex.DecodeString(col)
	if err != nil {
		return c, err
	}
	c.R = uint8(bytes[0])
	c.G = uint8(bytes[1])
	c.B = uint8(bytes[2])
	return
}
