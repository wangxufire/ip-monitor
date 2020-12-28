package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	for {
		ip, err := getExternalIP()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("fetch ip success %s", ip)
		ipFile := "/var/log/current-ip"
		_, err = os.Stat(ipFile)
		if os.IsNotExist(err) {
			f, err := os.Create(ipFile)
			if err != nil {
				fmt.Println(err)
				continue
			}
			defer f.Close()
			f.WriteString(ip)
			f.Sync()
		} else {
			f, err := os.OpenFile(ipFile, os.O_APPEND, 0666)
			if err != nil {
				fmt.Println(err)
				continue
			}
			defer f.Close()
			content := make([]byte, 10)
			f.Read(content)
			if ip != string(content) {
				url := fmt.Sprintf("https://api.day.app/2qT6qyWRNAYnYZx8mBsje7/ip-change/%s?isArchive=1&sound=minuet", ip)
				request, err := http.NewRequest("GET", url, nil)
				if err != nil {
					fmt.Println(err)
					continue
				}
				defer request.Body.Close()
				request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				client := &http.Client{}
				response, err := client.Do(request)
				if err != nil {
					fmt.Println(err)
					continue
				}
				defer response.Body.Close()
				f.Truncate(int64(len(content)))
				f.WriteString(ip)
				f.Sync()
			}
		}
		time.Sleep(10 * time.Minute)
	}
}

func getExternalIP() (string, error) {
	response, err := http.Get("http://ip.cip.cc")
	if err != nil {
		return "", errors.New("external IP fetch failed, detail:" + err.Error())
	}

	defer response.Body.Close()
	res := ""

	for {
		tmp := make([]byte, 32)
		n, err := response.Body.Read(tmp)
		if err != nil {
			if err != io.EOF {
				return "", errors.New("external IP fetch failed, detail:" + err.Error())
			}
			res += string(tmp[:n])
			break
		}
		res += string(tmp[:n])
	}
	return strings.TrimSpace(res), nil
}
