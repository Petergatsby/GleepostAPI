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
)

func main() {
	//Take user input
	var primary = flag.String("primary", "", "The app's primary colour")
	var leftNav = flag.String("leftnav", "", "The left nav-bar colour")
	var rightNav = flag.String("rightnav", "", "The right nav-bar colour")
	var navBackground = flag.String("navbar", "", "The nav-bar background colour")
	var wallTitle = flag.String("walltitle", "", "The campus wall title colour")
	var navBarTitle = flag.String("navtitle", "", "The general title colour")
	flag.Parse()
	args := flag.Args()
	filename := args[0]
	colours := make(map[string]color.RGBA)
	if *primary != "" {
		c, err := fromHex(*primary)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		colours["firstAutoColour"] = c
	}
	if *leftNav != "" {
		c, err := fromHex(*leftNav)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		colours["secondAutoColour"] = c
	}
	if *rightNav != "" {
		c, err := fromHex(*rightNav)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		colours["thirdAutoColour"] = c
	}
	if *navBackground != "" {
		c, err := fromHex(*navBackground)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		colours["fourthAutoColour"] = c
	}
	if *wallTitle != "" {
		c, err := fromHex(*wallTitle)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		colours["fifthAutoColour"] = c
	}
	if *navBarTitle != "" {
		c, err := fromHex(*navBarTitle)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		colours["sixthAutoColour"] = c

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
			switch {
			case i == 0:
				name = "firstAutoColour"
			case i == 1:
				name = "secondAutoColour"
			case i == 2:
				name = "thirdAutoColour"
			case i == 3:
				name = "fourthAutoColour"
			case i == 4:
				name = "fifthAutoColour"
			case i == 5:
				name = "sixthAutoColour"
			}
			colours[name] = c
		}
	}
	AppearanceHelper, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	for name, col := range colours {
		AppearanceHelper, err = setColour(AppearanceHelper, name, col)
	}
	err = ioutil.WriteFile(filename, AppearanceHelper, 0644)
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
