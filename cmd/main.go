package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("pass only one argument")
		os.Exit(1)
	}
	if os.Args[1] == "token" {
		Token()
		os.Exit(0)
	}
	if os.Args[1] == "download" {
		Download()
		os.Exit(0)
	}

	fmt.Println("please pass either token or download")
	os.Exit(1)
}
