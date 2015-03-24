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
	cols := make([]color.RGBA, 0)
	for _, arg := range args[1:] {
		c, err := fromHex(arg)
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		cols = append(cols, c)
	}
	AppearanceHelper, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	re := regexp.MustCompile("\\+ \\(UIColor \\*\\)(\\w+)AutoColour\n{\n    return \\[UIColor colorWithR:\\d+ withG:\\d+ andB:\\d+\\];\n}")
	replacer := func(cols []color.RGBA) func([]byte) []byte {
		i := 0
		return func([]byte) []byte {
			var name string
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
			result := []byte(colourM(cols[i], name))
			i++
			return result
		}
	}
	result := re.ReplaceAllFunc(AppearanceHelper, replacer(cols))
	err = ioutil.WriteFile(filename, result, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func colourM(c color.RGBA, name string) string {
	return fmt.Sprintf("+ (UIColor *)%s\n{\n    return [UIColor colorWithR:%d withG:%d andB:%d];\n}", name, c.R, c.G, c.B)
}

func colourH(name string) string {
	return fmt.Sprintf("+ (UIColor *)%s;\n", name)
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
