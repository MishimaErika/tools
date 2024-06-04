package main

import (
	"flag"
	"os"
	ping "zeus/parallelping"
)

var (
	f = flag.String("f", "", "文件名称")
)

func main() {
	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		return
	}

	if *f != "" {
		ping.ParallelPing(*f)
	}
}
