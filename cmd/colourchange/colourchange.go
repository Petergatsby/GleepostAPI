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

var (
	firstName  string = "firstColour"
	secondName string = "secondColour"
	thirdName  string = "thirdColour"
	fourthName string = "fourthColour"
	fifthName  string = "fifthColour"
)

func main() {
	//Take user input
	flag.Parse()
	args := flag.Args()
	if len(args) < 6 {
		fmt.Println("Not enough colours supplied.")
		os.Exit(-1)
	}
	filename := args[0]
	colours := make(map[string]color.RGBA)
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
