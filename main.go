package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/caiguanhao/finddeb/debpkgapi"
)

func main() {
	suite := flag.String("suite", "buster", "suite: jessie, stretch, buster, bullseye, sid ...")
	arch := flag.String("arch", "armhf", "architecture: amd64, arm64, armel, armhf, i386 ...")
	sarch := flag.String("arch4search", "", "architecture for search, can be 'any', default is the same as -arch")
	search := flag.Bool("search", false, "search packages by file name")
	list := flag.Bool("ls", false, "get file lists of packages")
	dir := flag.String("dir", "", "directory for downloaded files, defaults to <suite>-<arch>")
	mirror := flag.String("mirror", "ftp.debian.org", "prefer mirror that contains this string")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "finddeb [OPTIONS] [FILE NAMES]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Download files with the same names in the packages of "+
			"the selected arch and suite from Debian (https://packages.debian.org/)")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "[OPTIONS]")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "[EXAMPLE]")
		fmt.Fprintln(os.Stderr, "  finddeb libQt5Gui.so.5")
	}
	flag.Parse()

	args := flag.Args()

	searchArch := *sarch
	if searchArch == "" {
		searchArch = *arch
	}

	if *search {
		for i, arg := range args {
			if i > 0 {
				fmt.Println()
			}
			results, err := debpkgapi.SearchPackagesByFile(*suite, searchArch, arg)
			fmt.Println("Search results for:", arg)
			if err == nil {
				for _, result := range results {
					fmt.Printf("\t%s\n", result)
				}
			} else {
				fmt.Printf("\t%s\n", err)
			}
		}
		return
	}

	if *list {
		for i, arg := range args {
			if i > 0 {
				fmt.Println()
			}
			results, err := debpkgapi.GetFileList(*suite, *arch, arg)
			fmt.Println("List of files for:", arg)
			if err == nil {
				for _, result := range results {
					fmt.Printf("\t%s\n", result)
				}
			} else {
				fmt.Printf("\t%s\n", err)
			}
		}
		return
	}

	args, filesMap := parseArgs()

	dldir := *dir
	if dldir == "" {
		dldir = fmt.Sprintf("%s-%s", *suite, *arch)
	}

	var failedFiles []string
	for i, arg := range args {
		if i > 0 {
			fmt.Println()
		}
		var packages []string
		var links []string
	outer:
		for _, a := range filesMap[arg] {
			var err error
			fmt.Println("Search:", a)
			packages, err = debpkgapi.SearchPackagesByFile(*suite, searchArch, a)
			if err != nil {
				fmt.Printf("\t%s\n", err)
				continue
			}
			fmt.Printf("\tFound %d packages\n", len(packages))
			for _, p := range packages {
				fmt.Println("Download:", a)
				links, err = debpkgapi.GetDownloadLinks(*suite, *arch, p)
				if err != nil {
					fmt.Printf("\t%s\n", err)
					continue
				}
				fmt.Printf("\tFound %d links\n", len(links))
				if len(links) > 0 {
					break outer
				}
			}
		}
		var dllink string
		for _, link := range links {
			if strings.Contains(link, *mirror) {
				dllink = link
				break
			}
		}
		if dllink == "" && len(links) > 0 {
			dllink = links[0]
		}
		err := download(dldir, dllink, filesMap[arg])
		if err != nil {
			fmt.Printf("\tError: %s\n", err)
			failedFiles = append(failedFiles, filesMap[arg]...)
			continue
		}
	}
	if len(failedFiles) > 0 {
		fmt.Println()
		fmt.Println("Files failed to download:", strings.Join(failedFiles, " "))
		os.Exit(1)
	} else {
		fmt.Println()
		fmt.Println("Done!")
	}
}

func download(dldir, dllink string, fileNames []string) error {
	if dllink == "" {
		return errors.New("no links")
	}
	fmt.Printf("\tDownloading from %s\n", dllink)
	resp, err := http.Get(dllink)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	fmt.Printf("\tReceived %d bytes\n", len(b))
	br := bytes.NewReader(b)

	os.MkdirAll(dldir, 0755)
	var files []io.Writer
	var closes []func() error
	for _, name := range fileNames {
		dest := filepath.Join(dldir, name)
		f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		fmt.Printf("\tWritting to %s\n", dest)
		files = append(files, f)
		closes = append(closes, f.Close)
	}
	if len(files) == 0 {
		return errors.New("no files")
	}
	n, err := debpkgapi.GetFile(fileNames, br, io.MultiWriter(files...))
	for _, c := range closes {
		c()
	}
	if err == nil {
		fmt.Printf("\tDone (%d bytes written)\n", n)
	}
	return err
}

type StringByLen []string

func (x StringByLen) Len() int           { return len(x) }
func (x StringByLen) Less(i, j int) bool { return len(x[i]) > len(x[j]) }
func (x StringByLen) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func parseArgs() (args []string, filesMap map[string][]string) {
	filesMap = map[string][]string{}
	removeExt := regexp.MustCompile("\\.so(\\.\\d+){0,}$")
	for _, arg := range flag.Args() {
		noExt := removeExt.ReplaceAllString(arg, "")
		if _, ok := filesMap[noExt]; !ok {
			args = append(args, noExt)
		}
		filesMap[noExt] = append(filesMap[noExt], arg)
	}
	for key := range filesMap {
		sort.Sort(StringByLen(filesMap[key]))
	}
	return
}
