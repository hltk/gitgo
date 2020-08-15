package main

import (
	"os"
)

func fixpath(str *string) {
	if (*str)[len(*str)-1] != '/' {
		(*str) += "/"
	}
}

func makedir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}
}

func contentstolines(contents []byte, size int) []string {
	var lines = []string{""}

	for i := 0; i < size; i++ {
		c := contents[i]
		if c != '\n' {
			lines[len(lines)-1] += string(c)
		} else if i+1 != size {
			lines = append(lines, "")
		}
	}

	return lines
}
