package debpkgapi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func ExampleSearchPackagesByFile() {
	packages, err := SearchPackagesByFile("buster", "armhf", "libQt5Gui.so.5")
	fmt.Println(packages, err)
	// Output:
	// [libqt5gui5] <nil>
}

func ExampleSearchPackagesByName() {
	packages, err := SearchPackagesByName("libpulse-mainloop-glib")
	fmt.Println(packages, err)
	// Output:
	// [libpulse-mainloop-glib0 libpulse-mainloop-glib0-dbg libpulse-mainloop-glib0-dbgsym] <nil>
}

func ExampleGetFileList() {
	files, _ := GetFileList("buster", "armhf", "libavcodec58")
	fmt.Println(strings.Join(files, "\n"))
	// Output:
	// /usr/lib/arm-linux-gnueabihf/libavcodec.so.58
	// /usr/lib/arm-linux-gnueabihf/libavcodec.so.58.35.100
	// /usr/share/doc/libavcodec58/changelog.Debian.gz
	// /usr/share/doc/libavcodec58/changelog.gz
	// /usr/share/doc/libavcodec58/copyright
	// /usr/share/lintian/overrides/libavcodec58
}

func ExampleGetDownloadLinks() {
	links, _ := GetDownloadLinks("buster", "armhf", "libavcodec58")
	for _, link := range links {
		if strings.Contains(link, "ftp.debian.org") {
			fmt.Println(link)
		}
	}
	// Output:
	// http://ftp.debian.org/debian/pool/main/f/ffmpeg/libavcodec58_4.1.6-1~deb10u1_armhf.deb
}

func ExampleGetFile() {
	links, _ := GetDownloadLinks("buster", "armhf", "libwrap0")
	var dllink string
	for _, link := range links {
		if strings.Contains(link, "ftp.debian.org") {
			dllink = link
		}
	}
	resp, err := http.Get(dllink)
	if err != nil {
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	resp.Body.Close()
	br := bytes.NewReader(b)
	var buf bytes.Buffer
	GetFile([]string{"README.Debian"}, br, &buf)
	str := buf.String()
	fmt.Println(str[0:strings.Index(str, "\n")])
	// Output:
	// tcp_wrappers for Debian
}
