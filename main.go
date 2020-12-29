package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"
)

var (
	ipFile   string
	barkCode string
)

func init() {
	bark := flag.String("n", "2qT6qyWRNfAYYZx8sBsje7", "bark=${bark_device_code}")
	flag.Parse()
	barkCode = *bark
	current, err := user.Current()
	if err != nil {
		panic(err)
	}
	ipFile = current.HomeDir + string(os.PathSeparator) + ".current-ip"
}

func main() {
	for {
		ip, err := getExternalIP()
		if err != nil {
			log(err)
			sleep()
			continue
		}
		log(fmt.Sprintf("fetch ip success %s\n", ip))
		_, err = os.Stat(ipFile)
		if os.IsNotExist(err) {
			err = createIPFile(ipFile, ip)
			if err != nil {
				log(err)
			}
			sleep()
			continue
		}
		if err != nil {
			log(err)
			sleep()
			continue
		}
		err = compareAndRecordNewIP(ipFile, ip)
		if err != nil {
			log(err)
		}
		sleep()
	}
}

func sleep() {
	time.Sleep(10 * time.Minute)
}

func log(msg interface{}) {
	date := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("%s %+v \n", date, msg)
}

func createIPFile(ipFile, ip string) error {
	f, err := os.Create(ipFile)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(ip)
	f.Sync()
	return nil
}

func compareAndRecordNewIP(ipFile, ip string) error {
	f, err := os.OpenFile(ipFile, os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	buf, err := ioutil.ReadFile(f.Name())
	if err != nil {
		return err
	}
	oldIP := string(buf)
	if strings.Compare(ip, oldIP) != 0 {
		if err = notify(ip); err != nil {
			return err
		}
		f.Truncate(int64(len(buf)))
		f.WriteString(ip)
		f.Sync()
	}
	return nil
}

func notify(ip string) error {
	url := fmt.Sprintf("https://api.day.app/%s/ip-change/%s?isArchive=1&sound=birdsong", barkCode, ip)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	log("notify ip changed success")
	return nil
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
