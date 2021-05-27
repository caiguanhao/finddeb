package debpkgapi

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/blakesmith/ar"
	"github.com/ulikunitz/xz"
)

func SearchPackagesByFile(suite, arch, filename string) (packages []string, err error) {
	v := url.Values{}
	v.Set("mode", "filename")
	v.Set("suite", suite)
	v.Set("arch", arch)
	v.Set("searchon", "contents")
	v.Set("keywords", filename)
	url := "https://packages.debian.org/search?" + v.Encode()
	var res *http.Response
	res, err = http.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()
	var doc *goquery.Document
	doc, err = goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}
	doc.Find(".file").Each(func(_ int, s *goquery.Selection) {
		s.NextFiltered("td").Find("a").Each(func(_ int, a *goquery.Selection) {
			packages = appendIfMissing(packages, strings.TrimSpace(a.Text()))
		})
	})
	return
}

func SearchPackagesByName(name string) (results []string, err error) {
	v := url.Values{}
	v.Set("keywords", name)
	var res *http.Response
	res, err = http.Get("https://packages.debian.org/search?" + v.Encode())
	if err != nil {
		return
	}
	defer res.Body.Close()
	var doc *goquery.Document
	doc, err = goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}
	doc.Find("h3").Each(func(_ int, s *goquery.Selection) {
		results = append(results, strings.TrimSpace(strings.TrimPrefix(s.Text(), "Package")))
	})
	return
}

func GetFileList(suite, arch, packageName string) (files []string, err error) {
	var res *http.Response
	url := fmt.Sprintf("https://packages.debian.org/%s/%s/%s/filelist", suite, arch, packageName)
	res, err = http.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()
	var doc *goquery.Document
	doc, err = goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}
	files = strings.Split(doc.Find("pre").Text(), "\n")
	return
}

func GetDownloadLinks(suite, arch, packageName string) (links []string, err error) {
	var res *http.Response
	url := fmt.Sprintf("https://packages.debian.org/%s/%s/%s/download", suite, arch, packageName)
	res, err = http.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()
	var doc *goquery.Document
	doc, err = goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href := s.AttrOr("href", "")
		if strings.HasSuffix(href, ".deb") {
			links = append(links, href)
		}
	})
	return
}

func GetFile(targets []string, r io.ReadSeeker, w io.Writer) (int64, error) {
	arReader := ar.NewReader(r)
outer:
	for {
		header, err := arReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if header.Name != "data.tar.xz" {
			continue
		}
		xzr, err := xz.NewReader(arReader)
		if err != nil {
			return 0, err
		}
		tr := tar.NewReader(xzr)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, err
			}
			hasFile := false
			for _, target := range targets {
				if isFile(hdr.Typeflag) && filepath.Base(hdr.Name) == target {
					hasFile = true
					break
				}
			}
			if !hasFile {
				continue
			}
			if hdr.Linkname != "" {
				targets = []string{hdr.Linkname}
				_, err := r.Seek(0, io.SeekStart)
				if err != nil {
					return 0, err
				}
				arReader = ar.NewReader(r)
				continue outer
			}
			return io.Copy(w, tr)
		}
	}
	return 0, errors.New("no files matched in archive")
}

func appendIfMissing(items []string, i string) []string {
	for _, item := range items {
		if item == i {
			return items
		}
	}
	return append(items, i)
}

func isFile(flag byte) bool {
	switch flag {
	case tar.TypeReg, tar.TypeLink, tar.TypeSymlink:
		return true
	default:
		return false
	}
}
