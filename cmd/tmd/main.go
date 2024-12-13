package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go101.org/nstd"
	"go101.org/tmd/render"
)

func printUsage() {

	fmt.Printf(`Usage:
	%[1]v render [options] foo.tmd bar.tmd

Options:
	--full-html
		generate full page or not
	--support-custom-blocks
		support custom blocks or not
`,
		filepath.Base(os.Args[0]),
	)
}

func main() {
	flag.Parse()

	args := flag.Args()
	switch len(args) {
	case 0:
		printUsage()
		return
	case 1:
		switch sub := args[0]; sub {
		default:
			nstd.Printfln("Unkown sub-command: %s", sub)
			printUsage()
			return
		case "render":
			if len(args) == 1 {
				printUsage()
				return
			}
		}
	}

	renderer, err := render.NewRenderer()
	if err != nil {
		log.Fatal(err)
	}
	defer renderer.Destroy()

	const tmdExt = ".tmd"
	const htmlExt = ".html"

	var optionsDone = false
	var option_full_html = false
	var option_support_custom_blocks = false

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--") {
			if !optionsDone {
				switch arg[2:] {
				default:
					log.Fatalf("Unrecognized option: %s", arg[2:])
				case "full-html":
					option_full_html = true
				case "support-custom-blocks":
					option_support_custom_blocks = true
				}

				continue
			}
		} else {
			optionsDone = true
		}

		tmdData, err := os.ReadFile(arg)
		if err != nil {
			log.Printf("read TMD file [%s] error: %s", arg, err)
			continue
		}
		htmlData, err := renderer.Render(tmdData, option_full_html, option_support_custom_blocks)
		if err != nil {
			log.Printf("render file [%s] error: %s", arg, err)
			continue
		}

		var htmlFilepath string
		if strings.HasSuffix(strings.ToLower(arg), tmdExt) {
			htmlFilepath = arg[0:len(arg)-len(tmdExt)] + htmlExt
		} else {
			htmlFilepath = arg + htmlExt
		}
		err = os.WriteFile(htmlFilepath, htmlData, 0644)
		if err != nil {
			log.Printf("write HTML file [%s] error: %s", htmlFilepath, err)
			continue
		}

		fmt.Printf(`%s (%d bytes)
-> %s (%d bytes)
`, arg, len(tmdData), htmlFilepath, len(htmlData))
	}
}
