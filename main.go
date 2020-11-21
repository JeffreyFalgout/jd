package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	jd "github.com/josephburnett/jd/lib"
)

var patch = flag.Bool("p", false, "Patch mode")
var output = flag.String("o", "", "Output file")
var set = flag.Bool("set", false, "Arrays as sets")
var mset = flag.Bool("mset", false, "Arrays as multisets")
var setkeys = flag.String("setkeys", "", "Keys to identify set objects")
var port = flag.Int("port", 0, "Serve web UI on port.")
var yaml = flag.Bool("yaml", false, "Read and write YAML instead of JSON.")

func main() {
	flag.Parse()
	if *port != 0 {
		serveWeb(strconv.Itoa(*port))
		return
	}
	metadata, err := parseMetadata()
	if err != nil {
		log.Fatalf(err.Error())
	}
	var a, b string
	switch len(flag.Args()) {
	case 1:
		a = readFile(flag.Arg(0))
		b = readStdin()
	case 2:
		a = readFile(flag.Arg(0))
		b = readFile(flag.Arg(1))
	default:
		printUsageAndExit()
	}
	patchFn := patchJson
	diffFn := diffJson
	if *yaml {
		patchFn = patchYaml
		diffFn = diffYaml
	}
	if *patch {
		patchFn(a, b, metadata)
	} else {
		diffFn(a, b, metadata)
	}
}

func serveWeb(port string) error {
	fs := http.FileServer(http.Dir("./web/assets"))
	http.Handle("/", fs)
	log.Printf("Listening on :%v...", port)
	return http.ListenAndServe(":"+port, nil)
}

func parseMetadata() ([]jd.Metadata, error) {
	metadata := make([]jd.Metadata, 0)
	if *set {
		metadata = append(metadata, jd.SET)
	}
	if *mset {
		metadata = append(metadata, jd.MULTISET)
	}
	if *setkeys != "" {
		keys := make([]string, 0)
		ks := strings.Split(*setkeys, ",")
		for _, k := range ks {
			trimmed := strings.TrimSpace(k)
			if trimmed == "" {
				return nil, fmt.Errorf("Invalid set key: %v", k)
			}
			keys = append(keys, trimmed)
		}
		metadata = append(metadata, jd.Setkeys(keys...))
	}
	return metadata, nil
}

func printUsageAndExit() {
	for _, line := range []string{
		``,
		`Usage: jd [OPTION]... FILE1 [FILE2]`,
		`Diff and patch JSON files.`,
		``,
		`Prints the diff of FILE1 and FILE2 to STDOUT.`,
		`When FILE2 is omitted the second input is read from STDIN.`,
		`When patching (-p) FILE1 is a diff.`,
		``,
		`Metadata:`,
		`  -p        Apply patch FILE1 to FILE2 or STDIN.`,
		`  -o=FILE3  Write to FILE3 instead of STDOUT.`,
		`  -set      Treat arrays as sets.`,
		`  -mset     Treat arrays as multisets (bags).`,
		`  -yaml     Read and write YAML instead of JSON.`,
		``,
		`Examples:`,
		`  jd a.json b.json`,
		`  cat b.json | jd a.json`,
		`  jd -o patch a.json b.json; jd patch a.json`,
		`  jd -set a.json b.json`,
		``,
	} {
		fmt.Println(line)
	}
	os.Exit(1)
}

func diffJson(a, b string, metadata []jd.Metadata) {
	aNode, err := jd.ReadJsonString(a)
	if err != nil {
		log.Fatalf(err.Error())
	}
	bNode, err := jd.ReadJsonString(b)
	if err != nil {
		log.Fatalf(err.Error())
	}
	diff := aNode.Diff(bNode, metadata...)
	if *output == "" {
		fmt.Print(diff.Render())
	} else {
		ioutil.WriteFile(*output, []byte(diff.Render()), 0644)
	}
}

func diffYaml(a, b string, metadata []jd.Metadata) {
	aNode, err := jd.ReadYamlString(a)
	if err != nil {
		log.Fatalf(err.Error())
	}
	bNode, err := jd.ReadYamlString(b)
	if err != nil {
		log.Fatalf(err.Error())
	}
	diff := aNode.Diff(bNode, metadata...)
	if *output == "" {
		fmt.Print(diff.Render())
	} else {
		ioutil.WriteFile(*output, []byte(diff.Render()), 0644)
	}
}

func patchJson(p, a string, metadata []jd.Metadata) {
	diff, err := jd.ReadDiffString(p)
	if err != nil {
		log.Fatalf(err.Error())
	}
	aNode, err := jd.ReadJsonString(a)
	if err != nil {
		log.Fatalf(err.Error())
	}
	bNode, err := aNode.Patch(diff)
	if err != nil {
		log.Fatalf(err.Error())
	}
	if *output == "" {
		fmt.Print(bNode.Json(metadata...))
	} else {
		ioutil.WriteFile(*output, []byte(bNode.Json(metadata...)), 0644)
	}
}

func patchYaml(p, a string, metadata []jd.Metadata) {
	diff, err := jd.ReadDiffString(p)
	if err != nil {
		log.Fatalf(err.Error())
	}
	aNode, err := jd.ReadYamlString(a)
	if err != nil {
		log.Fatalf(err.Error())
	}
	bNode, err := aNode.Patch(diff)
	if err != nil {
		log.Fatalf(err.Error())
	}
	if *output == "" {
		fmt.Print(bNode.Yaml(metadata...))
	} else {
		ioutil.WriteFile(*output, []byte(bNode.Yaml(metadata...)), 0644)
	}
}

func readFile(filename string) string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return string(bytes)
}

func readStdin() string {
	r := bufio.NewReader(os.Stdin)
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return string(bytes)
}
