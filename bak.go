package main

import (
	"fmt"
	"regexp"
	"runtime"
)

// GetZabbixAgentLink returns the zabbix agent link
func GetZabbixAgentLink(links []string) string {
	var zaLinks []string
	// Filter links that contain the keyword zabbix-agent or zabbix_agent
	for i := range links {
		if IsContainsOr(links[i], []string{"zabbix-agent", "zabbix_agent"}) {
			zaLinks = append(zaLinks, links[i])
		}
	}
	// OS type,windows or linux
	ot := runtime.GOOS
	// System architecture
	oa := runtime.GOARCH
	var avaLinks []string

	switch ot {
	case "windows":
		for i := range zaLinks {
			if IsContainsOr(links[i], []string{"amd64"}) && IsContainsAnd(zaLinks[i], []string{"win"}) {
				avaLinks = append(avaLinks, zaLinks[i])
			} else {
				Logger("ERROR", fmt.Sprintf("unknown OS arch:%s", oa))
			}
		}
	case "linux":
		for i := range zaLinks {
			if oa == "amd64" {
				if IsContainsOr(zaLinks[i], []string{"amd64", "x86_64"}) && IsContainsAnd(zaLinks[i], []string{"linux"}) {
					avaLinks = append(avaLinks, zaLinks[i])
				}
			} else if oa == "386" {
				if IsContainsOr(zaLinks[i], []string{"386"}) && IsContainsAnd(zaLinks[i], []string{"linux"}) {
					avaLinks = append(avaLinks, zaLinks[i])
				}
			} else {
				Logger("ERROR", fmt.Sprintf("unknown OS arch:%s", oa))
			}
		}
	default:
		Logger("ERROR", fmt.Sprintf("unknown OS type:%s", ot))
	}
	return avaLinks[len(avaLinks)-1]
}

// GetZabbixAgentPackageName Filter zabbix installation package names
func GetZabbixAgentPackageName(filenames []string) (string, error) {
	var avaFilenames []string
	switch runtime.GOOS {
	case "linux":
		reg, err := regexp.Compile(`zabbix.*agent.*linux.*\.tar\.gz`)
		if err != nil {
			return "", err
		}
		for _, filename := range filenames {
			if reg.MatchString(filename) {
				avaFilenames = append(avaFilenames, filename)
			}
		}
	case "windows":
		reg, err := regexp.Compile(`zabbix.*agent.*win.*\.zip`)
		if err != nil {
			return "", err
		}
		for _, filename := range filenames {
			if reg.MatchString(filename) {
				avaFilenames = append(avaFilenames, filename)
			}
		}
	default:
		return "", fmt.Errorf("unsupported operating system")
	}
	if len(avaFilenames) == 0 {
		return "", fmt.Errorf("no package found")
	}
	return avaFilenames[len(avaFilenames)-1], nil
}
