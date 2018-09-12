package main

import (
	// 需在此处添加代码。[1]
	"flag"
	"fmt"
)

func init() {
	// 需在此处添加代码。[2]
}

func main() {
	// 需在此处添加代码。[3]
	var name = *flag.String("name", "everyone", "The greeting object.")
	// name := *flag.String("name", "everyone", "The greeting object.")

	flag.Parse()
	fmt.Printf("Hello, %s!\n", name)
}
