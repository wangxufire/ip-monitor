package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ipFile    string
	period    uint
	bark      string
	domain    string
	secretID  string
	secretKey string
)

func init() {
	flag.StringVar(&bark, "bark", "", "-bark ${bark_device_code} see https://github.com/Finb/Bark")
	flag.UintVar(&period, "period", 600, "-period 600 unit is second")
	flag.StringVar(&domain, "domain", "", "-domain domain")
	flag.StringVar(&secretID, "secretId", "", "-secretId tencent cloud api SecretId")
	flag.StringVar(&secretKey, "secretKey", "", "-secretKey tencent cloud api secretKey")
	flag.Parse()

	if bark == "" || domain == "" {
		flag.Usage()
		os.Exit(0)
	}

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
		log(fmt.Sprintf("fetch ip success %s", ip))
		_, err = os.Stat(ipFile)
		if err != nil {
			if os.IsNotExist(err) {
				err = createIPFile(ipFile, ip)
			}
			if err != nil {
				log(err)
			}
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
	time.Sleep(time.Duration(period) * time.Second)
}

func log(msg interface{}) {
	date := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("%s %+v\n", date, msg)
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
	f, err := os.OpenFile(ipFile, os.O_RDWR, os.ModePerm)
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
		if err = updateDNS(ip); err != nil {
			return err
		}
		if err = notify(ip); err != nil {
			return err
		}
		if err = f.Truncate(0); err != nil {
			return err
		}
		if _, err = f.Seek(0, 0); err != nil {
			return err
		}
		if _, err = f.WriteString(ip); err != nil {
			return err
		}
		if err = f.Sync(); err != nil {
			return err
		}
	}
	return nil
}

func notify(ip string) error {
	if bark == "" {
		return nil
	}
	url := fmt.Sprintf("https://api.day.app/%s/ip-change/%s?isArchive=1&sound=birdsong", bark, ip)
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

func sign(method, signType string, params map[string]string) (string, error) {
	timestamp := time.Now().Unix()
	// 添加公共部分
	params["Timestamp"] = strconv.FormatInt(timestamp, 10)
	rand.Seed(timestamp)
	params["Nonce"] = fmt.Sprintf("%v", rand.Int())
	params["SecretId"] = secretID

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	// 对参数的下标进行升序排序
	sort.Strings(keys)

	var requestParams string
	for _, k := range keys {
		requestParams += k + "=" + params[k] + "&"
	}

	requestParams = requestParams[0 : len(requestParams)-1]

	var mac hash.Hash
	switch signType {
	case "HmacSHA1":
		mac = hmac.New(sha1.New, []byte(secretKey))
	case "HmacSHA256":
		mac = hmac.New(sha256.New, []byte(secretKey))
	default:
		return "", errors.New("加密参数错误")
	}

	mac.Write([]byte(strings.ToUpper(method) + "cns.api.qcloud.com/v2/index.php?" + requestParams))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	sign = url.QueryEscape(sign)

	return requestParams + "&Signature=" + sign, nil
}

func requestTencentCloud(URL string) (map[string]interface{}, error) {
	res, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		log(body)
		return nil, err
	}
	return response, nil
}

func cnsRecordList(subDomain string) (map[string]interface{}, error) {
	apiURL := "https://cns.api.qcloud.com/v2/index.php?"
	params := map[string]string{
		"Action":    "RecordList",
		"domain":    domain,
		"subDomain": subDomain,
	}
	paramsStr, err := sign("get", "HmacSHA1", params)
	if err != nil {
		return nil, err
	}
	return requestTencentCloud(apiURL + paramsStr)
}

func cnsRecordModify(recordID, subDomain, value string) error {
	apiURL := "https://cns.api.qcloud.com/v2/index.php?"
	params := map[string]string{
		"Action":     "RecordModify",
		"domain":     domain,
		"recordId":   recordID,
		"subDomain":  subDomain,
		"recordType": "A",
		"recordLine": "默认",
		"value":      value,
		"ttl":        "600",
	}
	paramsStr, err := sign("get", "HmacSHA1", params)
	if err != nil {
		return err
	}
	res, err := requestTencentCloud(apiURL + paramsStr)
	if err != nil {
		return err
	}
	log(fmt.Sprintf("update sub-domain %s: %s", subDomain, res["codeDesc"].(string)))
	return nil
}

func updateDNS(ip string) error {
	if domain == "" {
		return nil
	}
	err := updateSub("@", ip)
	if err != nil {
		return err
	}
	err = updateSub("www", ip)
	if err != nil {
		return err
	}
	return nil
}

func updateSub(subDomain, ip string) error {
	resp, err := cnsRecordList(subDomain)
	if err != nil {
		return err
	}
	if resp["code"].(float64) != 0 {
		return errors.New("get dns record list failed")
	}
	for _, v := range resp["data"].(map[string]interface{})["records"].([]interface{}) {
		record := v.(map[string]interface{})
		if record["type"].(string) == "A" {
			id := record["id"].(float64)
			recordID := strconv.FormatFloat(id, 'f', -1, 64)
			err := cnsRecordModify(recordID, subDomain, ip)
			if err != nil {
				return err
			}
			log("update dns success")
			break
		}
	}
	return nil
}
