package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go101.org/nstd"
	"go101.org/tmd/lib"
)

// ToDo: lib version and the command version might be different.
func printUsage(version []byte) {

	fmt.Printf(`TapirMD toolset v%s

Usages:
	%[2]v gen [gen-options] foo.tmd bar.tmd
	%[2]v fmt foo.tmd bar.tmd

gen-options:
	--full-html
		generate full page or not
	--support-custom-blocks
		support custom blocks or not
`,
		version,
		filepath.Base(os.Args[0]),
	)
}

func main() {
	tmdLib, err := lib.NewTmdLib()
	if err != nil {
		log.Fatal(err)
	}
	defer tmdLib.Destroy()

	libVersion, err := tmdLib.Version()
	if err != nil {
		log.Fatal(err)
	}

	flag.Parse()

	args := flag.Args()
	switch len(args) {
	case 0:
		printUsage(libVersion)
		return
	default:
		switch sub := args[0]; sub {
		default:
			nstd.Printfln("Unkown sub-command: %s", sub)
			printUsage(libVersion)
		case "gen":
			if len(args) == 1 {
				printUsage(libVersion)
				return
			}
			generateHTML(tmdLib, args[1:])
		case "fmt":
			if len(args) == 1 {
				printUsage(libVersion)
				return
			}
			formatTMD(tmdLib, args[1:])
		}
	}
}

func generateHTML(tmdLib *lib.TmdLib, args []string) {
	const tmdExt = ".tmd"
	const htmlExt = ".html"

	var optionsDone = false
	var option_full_html = false
	var option_support_custom_blocks = false

	for _, arg := range args {
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

		var tmdFilePath = arg
		tmdData, err := os.ReadFile(tmdFilePath)
		if err != nil {
			log.Printf("read TMD file [%s] error: %s", tmdFilePath, err)
			continue
		}
		htmlData, err := tmdLib.GenerateHtmlFromTmd(tmdData, option_full_html, option_support_custom_blocks)
		if err != nil {
			log.Printf("geneate HTML file [%s] error: %s", tmdFilePath, err)
			continue
		}

		var htmlFilepath string
		if strings.HasSuffix(strings.ToLower(tmdFilePath), tmdExt) {
			htmlFilepath = tmdFilePath[0:len(tmdFilePath)-len(tmdExt)] + htmlExt
		} else {
			htmlFilepath = tmdFilePath + htmlExt
		}
		err = os.WriteFile(htmlFilepath, htmlData, 0644)
		if err != nil {
			log.Printf("write HTML file [%s] error: %s", htmlFilepath, err)
			continue
		}

		fmt.Printf(`%s (%d bytes)
-> %s (%d bytes)
`, tmdFilePath, len(tmdData), htmlFilepath, len(htmlData))
	}
}

func formatTMD(tmdLib *lib.TmdLib, args []string) {
	for _, arg := range args {
		var tmdFilePath = arg
		fileInfo, err := os.Stat(tmdFilePath)
		if err != nil {
			log.Printf("stat TMD file [%s] error: %s", tmdFilePath, err)
			continue
		}
		tmdData, err := os.ReadFile(tmdFilePath)
		if err != nil {
			log.Printf("read TMD file [%s] error: %s", tmdFilePath, err)
			continue
		}
		// fileInfo.Size() == len(tmdData)
		formatData, err := tmdLib.FormatTmd(tmdData)
		if err != nil {
			log.Printf("format TMD file [%s] error: %s", tmdFilePath, err)
			continue
		}

		if formatData != nil {
			err = os.WriteFile(tmdFilePath, formatData, fileInfo.Mode())
			if err != nil {
				log.Printf("write TMD file [%s] error: %s", tmdFilePath, err)
				continue
			}

			fmt.Printf("%s\n", tmdFilePath)
		}
	}
}
