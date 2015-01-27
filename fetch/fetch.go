package fetch

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/axgle/mahonia"
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const (
	MAX_ROUTINE   int    = 200
	IMAGE_DIR     string = `./static/image/`
	BAIDU_PAGENUM int    = 1
	//BAIDU_URL     string = `http://image.baidu.com/i?tn=resultjson_com&ipn=rj&ct=201326592&cl=2&lm=-1&st=-1&fm=index&fr=&sf=1&fmq=&pv=&ic=0&nc=1&z=&se=1&showtab=0&fb=0&width=&height=&face=0&istype=2&ie=gbk&word=%s&oe=utf-8&rn=60&pn=%d`
	BAIDU_URL      string = `http://image.baidu.com/i?tn=baiduimagejson&ipn=r&ct=201326592&cl=2&lm=-1&st=-1&fm=result&fr=&sf=1&fmq=1421656094748_R&pv=&ic=0&nc=1&z=&se=1&showtab=0&fb=0&width=&height=&face=0&istype=2&ie=utf-8&word=%s&x=%d`
	SOUGOU_PAGENUM int    = 2
	SOUGOU_URL     string = `http://pic.sogou.com/pics?query=%s&mood=0&picformat=0&mode=1&di=2&w=05009900&dr=1&_asf=pic.sogou.com&_ast=%d&start=%d&reqType=ajax&tn=0&reqFrom=result`
	QIHU_PAGENUM   int    = 2
	QIHU_URL       string = `http://image.haosou.com/j?q=%s&src=srp&sn=%d&pn=30`
)

var WWW_URL string
var MyClient *http.Client

//图片结构定义
type Item struct {
	Thumbnail string
	Img       string
	Desc      string
	From      string
	Width     string
	Height    string
}

func init() {
	addr := GetIP()
	WWW_URL = `http://` + addr + `:8888/`
	MyClient = &http.Client{
		CheckRedirect: nil, //RedirectFunc,
	}
}

func GetIP() string {
	conn, err := net.Dial("udp", "google.com:80")
	defer conn.Close()
	if err != nil {
		return `122.11.33.188`
	}
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}

func Fetch(key string, from string) (*[]*Item, error) {
	if key == "" {
		return nil, errors.New("Key is nil " + from)
	}
	switch from {
	case "baidu":
		return FetchFromBaidu(key)
	case "qihu":
		return FetchFromQihu(key)
	case "sougou":
		return FetchFromSougou(key)
	default:
		return nil, errors.New("from error " + from)
	}
}

func FetchFromBaidu(key string) (*[]*Item, error) {
	ukey := EncodeKey(key)
	surl := GetBaiduUrl(ukey)
	ItemList := make([]*Item, 0, BAIDU_PAGENUM*60+1)

	for _, url := range surl {
		//fmt.Println(url)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Baidu: error request")
			return nil, errors.New("Baidu:" + err.Error())
		}
		response, err := MyClient.Do(req)
		if err != nil {
			fmt.Println("Baidu: bad request")
			return nil, errors.New("Baidu:" + err.Error())
		}
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		jdata, err := simplejson.NewJson(data)
		if err != nil {
			continue
		}
		ParseBaiduResponse(&ItemList, jdata)
	}
	DownloadImage(&ItemList, "baidu")
	return &ItemList, nil
}

func DownloadImage(ItemList *[]*Item, from string) {
	ch := make(chan bool, MAX_ROUTINE)
	for _, val := range *ItemList {
		img := val.Img
		thum := val.Thumbnail
		val.Img = EncodeImgPath(img)
		val.Thumbnail = EncodeImgPath(thum)
		ch <- true
		go RealDownload(img, ch, from)
		ch <- true
		go RealDownload(thum, ch, from)
	}
}

func RealDownload(url string, ch chan bool, from string) {
	defer func() {
		<-ch
	}()
	if url == "" {
		return
	}
	req, err := GetRequest("GET", url, from)
	if err != nil {
		fmt.Println("GetRequest error")
		return
	}
	//client := &http.Client{}
	resp, err := MyClient.Do(req)
	if err != nil {
		fmt.Println("请求失败：", err, url)
		return
	}
	defer resp.Body.Close()
	/**
	content := resp.Header.Get("Content-Type")
	if !strings.Contains(content, "image") {
		fmt.Println("返回的不是img", url)
		return
	}
	*/
	data, _ := ioutil.ReadAll(resp.Body)
	fname := IMAGE_DIR + GetMd5String(filepath.Base(url)) + filepath.Ext(url)
	ioutil.WriteFile(fname, data, 0755)
}
func ParseBaiduResponse(ItemList *[]*Item, jsons *simplejson.Json) {
	jdata, _ := jsons.Get("data").Array()
	for _, value := range jdata {
		val := value.(map[string]interface{})
		if len(val) == 0 {
			break
		}
		item := new(Item)
		item.Desc = val["fromPageTitleEnc"].(string)
		item.From = val["fromURLHost"].(string)
		item.Img = val["objURL"].(string)
		item.Thumbnail = val["thumbURL"].(string)
		//fmt.Println(item.Img)
		item.Width = val["width"].(json.Number).String()
		item.Height = val["height"].(json.Number).String()
		*ItemList = append(*ItemList, item)
	}
}

func FetchFromQihu(key string) (*[]*Item, error) {
	ukey := EncodeKey(key)
	surl := GetQihuUrl(ukey)
	ItemList := make([]*Item, 0, QIHU_PAGENUM*30+1)
	for _, url := range surl {
		//fmt.Println(url)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Qihu: error request")
			return nil, errors.New("Qihu:" + err.Error())
		}
		response, err := MyClient.Do(req)
		if err != nil {
			fmt.Println("Qihu: bad request")
			return nil, errors.New("Qihu:" + err.Error())
		}
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		jdata, err := simplejson.NewJson(data)
		if err != nil {
			continue
		}
		ParseQihuResponse(&ItemList, jdata)
	}
	DownloadImage(&ItemList, "qihu")
	return &ItemList, nil
}

func ParseQihuResponse(ItemList *[]*Item, jsons *simplejson.Json) {
	jdata, _ := jsons.Get("list").Array()
	for _, value := range jdata {
		val := value.(map[string]interface{})
		if len(val) == 0 {
			break
		}
		item := new(Item)
		item.Desc = val["title"].(string)
		item.From = val["dspurl"].(string)
		item.Thumbnail = val["thumb"].(string)
		item.Img = val["img"].(string)
		item.Width = val["width"].(string)
		item.Height = val["height"].(string)
		*ItemList = append(*ItemList, item)
	}
}
func FetchFromSougou(key string) (*[]*Item, error) {
	ukey := EncodeKey(key)
	surl := GetSougouUrl(ukey)
	ItemList := make([]*Item, 0, SOUGOU_PAGENUM*48+1)
	for _, url := range surl {
		//fmt.Println(url)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Sougou: error request")
			return nil, errors.New("Sougou:" + err.Error())
		}
		response, err := MyClient.Do(req)
		if err != nil {
			fmt.Println("Sougou: bad request")
			return nil, errors.New("Sougou:" + err.Error())
		}
		defer response.Body.Close()
		data, _ := ioutil.ReadAll(response.Body)
		jdata, err := simplejson.NewJson(data)
		if err != nil {
			continue
		}
		ParseSougouResponse(&ItemList, jdata)
	}
	DownloadImage(&ItemList, "sougou")
	return &ItemList, nil
}
func ParseSougouResponse(ItemList *[]*Item, jsons *simplejson.Json) {
	jdata, _ := jsons.Get("items").Array()
	for _, value := range jdata {
		val := value.(map[string]interface{})
		if len(val) == 0 {
			break
		}
		item := new(Item)
		item.Desc = val["title"].(string)
		//item.From = val[""]
		item.Thumbnail = val["thumbUrl"].(string)
		item.Img = val["pic_url_noredirect"].(string)
		item.Height = val["width"].(string)
		item.Width = val["height"].(string)
		*ItemList = append(*ItemList, item)
	}
}
func GetBaiduUrl(key string) []string {
	var surl []string = make([]string, 0, BAIDU_PAGENUM)
	for i := 0; i < BAIDU_PAGENUM; i++ {
		url := fmt.Sprintf(BAIDU_URL, key, i*60)
		surl = append(surl, url)
	}
	return surl
}

func GetSougouUrl(key string) []string {
	var surl []string = make([]string, 0, SOUGOU_PAGENUM)
	for i := 0; i < SOUGOU_PAGENUM; i++ {
		surl = append(surl, fmt.Sprintf(SOUGOU_URL, key, time.Now().Unix(), i*48))
	}
	return surl
}
func GetQihuUrl(key string) []string {
	var surl []string = make([]string, 0, QIHU_PAGENUM)
	for i := 0; i < QIHU_PAGENUM; i++ {
		surl = append(surl, fmt.Sprintf(QIHU_URL, key, i*30))
	}
	return surl
}

func EncodeImgPath(path string) string {
	result := WWW_URL + "view?imgID=" + GetMd5String(filepath.Base(path)) + filepath.Ext(path)
	return result
}

func GetMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func GetRequest(method, url, from string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println("Gen wrong request:", url)
		return nil, err
	}
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	//req.Header.Add("Accept-Encoding", "gzip")
	req.Header.Add("Accept-Language", "zh-CN")
	req.Header.Add("Host", GetHostString(from))
	req.Header.Add("Referer", GetRefererString(from))
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Maxthon/4.4.3.2000 Chrome/30.0.1599.101 Safari/537.36")
	return req, nil
}

func GetHostString(from string) string {
	switch from {
	case "baidu":
		return `image.baidu.com`
	case "qihu":
		return `image.haosou.com`
	case "sougou":
		return `pic.sogou.com`
	default:
		return `image.baidu.com`
	}
}

func GetRefererString(from string) string {
	switch from {
	case "baidu":
		return `http://image.baidu.com`
	case "qihu":
		return `http://image.haosou.com`
	case "sougou":
		return `http://pic.sogou.com`
	default:
		return `http://baidu.com`
	}
}
func EncodeKey(key string) string {
	enc := mahonia.NewEncoder("gb18030")
	r := enc.ConvertString(key)
	return url.QueryEscape(r)
}

//重定向
func RedirectFunc(req *http.Request, via []*http.Request) error {
	return nil
}
