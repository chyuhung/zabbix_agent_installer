package mynet

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"zabbix_agent_installer/mylog"

	"golang.org/x/net/html"
)

// 判断ip是否合规
func IsIPv4(ipv4 string) bool {
	ip := net.ParseIP(ipv4)
	if ip == nil {
		return false
	}
	ip = ip.To4()
	return ip != nil
}

// 测试ip是否可达
func IsUnreachable(ipv4 string, port string) bool {
	addr := net.JoinHostPort(ipv4, port)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return true
	}
	defer conn.Close()
	return false
}

// GetMainIP gets the IP address of the host.
func GetMainIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return strings.Split(localAddr.String(), ":")[0], nil
}

// source:http://www.codebaoku.com/it-go/it-go-168428.html
func visit(links []string, n *html.Node) []string {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				links = append(links, a.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links = visit(links, c)
	}
	return links
}

// GetLinks returns the name of the package
func GetLinks(url string) ([]string, error) {
	var links []string
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, _ := html.Parse(resp.Body)
	for _, link := range visit(nil, doc) {
		links = append(links, url+link)
	}
	return links, nil
}

// 下载安装包
func DownloadPackage(url string, saveAbsPath string) string {
	resp, err := http.Get(url)
	if err != nil {
		mylog.Logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			mylog.Logger("ERROR", err.Error())
			os.Exit(1)
		}
	}()
	// 创建文件，从url中读取文件名
	filename := path.Base(url)
	mylog.Logger("INFO", fmt.Sprintf("starting to download %s", filename))
	out, err := os.OpenFile(filepath.Join(saveAbsPath, filename), os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		mylog.Logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	defer func() {
		err := out.Close()
		if err != nil {
			mylog.Logger("ERROR", err.Error())
			os.Exit(1)
		}
	}()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		mylog.Logger("ERROR", "download package failed "+err.Error())
		os.Exit(1)
	}
	mylog.Logger("INFO", fmt.Sprintf("%s was saved to %s", filename, saveAbsPath))
	mylog.Logger("INFO", "Download successful")
	return filename
}
