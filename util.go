package main

import (
	"io"
	"log"
	"os"
	"strings"
)

func writetofile(file *os.File, str string) {
	_, err := io.WriteString(file, str)
	if err != nil {
		log.Fatal(err)
	}
}

func openfile(str string) *os.File {
	file, err := os.Create(str)
	if err != nil {
		log.Fatal(err)
	}
	return file

}

func closefile(file *os.File) {
	if err := file.Sync(); err != nil {
		log.Fatal(err)
	}

	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}

func fixpath(str *string) {
	if (*str)[len(*str)-1] != '/' {
		(*str) += "/"
	}
}

func cleanname(name string) string {
	name = strings.TrimSuffix(name, ".git")

	lastslash := strings.LastIndex(name, "/")

	if lastslash != -1 {
		name = name[lastslash+1:]
	}

	return name
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

func capcommitsummary(summary string) string {
	if len(summary) > Config.MaxSummaryLen {
		summary = summary[:Config.MaxSummaryLen-3] + "..."
	}
	return summary
}
