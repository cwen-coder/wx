package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/go-martini/martini"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"wx/haddle"
)

const (
	token             = "wxcwenyin"
	appID             = "wx1abd680617d17f5f"
	appSecret         = "7aa1af925a4483b126be488502c5e7db"
	weixinHost        = "https://api.weixin.qq.com/cgi-bin"
	weixinQRScene     = "https://api.weixin.qq.com/cgi-bin/qrcode"
	weixinShowQRScene = "https://mp.weixin.qq.com/cgi-bin/showqrcode"
	weixinShortURL    = "https://api.weixin.qq.com/cgi-bin/shorturl"
	weixinUserInfo    = "https://api.weixin.qq.com/cgi-bin/user/info"
	weixinFileURL     = "http://file.api.weixin.qq.com/cgi-bin/media"
	retryMaxN         = 1
)

type TextRequestBody struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string
	FromUserName string
	CreateTime   time.Duration
	MsgType      string
	Content      string
	MsgId        int
}

type ImgResponseBody struct {
	XMLName      xml.Name `xml:"xml"`
	FromUserName CDATAText
	ToUserName   CDATAText
	MsgType      CDATAText
	MediaId      CDATAText
	CreateTime   time.Duration
}

type TextResponseBody struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATAText
	FromUserName CDATAText
	CreateTime   time.Duration
	MsgType      CDATAText
	Content      CDATAText
}

/*
	type CDATAText struct {
			Text []byte `xml:",innerxml"`
		}
*/

type accessToken struct {
	token   string
	expires time.Time
}

type CDATAText struct {
	Text string `xml:",innerxml"`
}

type Weixin struct {
	token string
	//routes    []*route
	tokenChan chan accessToken
	//ticketChan chan jsApiTicket
	userData  interface{}
	appId     string
	appSecret string
}

type response struct {
	ErrorCode    int    `json:"errcode,omitempty"`
	ErrorMessage string `json:"errmsg,omitempty"`
}

func (wx *Weixin) UploadMedia(mediaType string, filename string, reader io.Reader) (string, error) {
	return uploadMedia(wx.tokenChan, mediaType, filename, reader)
}

func (wx *Weixin) UploadMediaFromFile(mediaType string, fp string) (string, error) {
	file, err := os.Open(fp)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return wx.UploadMedia(mediaType, filepath.Base(fp), file)
}

func (wx *Weixin) PostText(touser string, text string) error {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	}
	msg.ToUser = touser
	msg.MsgType = "text"
	msg.Text.Content = text
	return postMessage(wx.tokenChan, &msg)
}

func postMessage(c chan accessToken, msg interface{}) error {
	data, err := marshal(msg)
	if err != nil {
		return err
	}
	_, err = postRequest(weixinHost+"/message/custom/send?access_token=", c, data)
	return err
}

func marshal(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err == nil {
		data = bytes.Replace(data, []byte("\\u003c"), []byte("<"), -1)
		data = bytes.Replace(data, []byte("\\u003e"), []byte(">"), -1)
		data = bytes.Replace(data, []byte("\\u0026"), []byte("&"), -1)
	}
	return data, err
}

func postRequest(reqURL string, c chan accessToken, data []byte) ([]byte, error) {
	for i := 0; i < retryMaxN; i++ {
		token := <-c
		if time.Since(token.expires).Seconds() < 0 {
			r, err := http.Post(reqURL+token.token, "application/json; charset=utf-8", bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			defer r.Body.Close()
			reply, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			var result response
			if err := json.Unmarshal(reply, &result); err != nil {
				return nil, err
			}
			switch result.ErrorCode {
			case 0:
				return reply, nil
			case 42001: // access_token timeout and retry
				continue
			default:
				return nil, errors.New(fmt.Sprintf("WeiXin send post request reply[%d]: %s", result.ErrorCode, result.ErrorMessage))
			}
		}
	}
	return nil, errors.New("WeiXin post request too many times:" + reqURL)
}

func New(token1 string, appid1 string, secret1 string) *Weixin {
	wx := &Weixin{}
	wx.token = token1
	wx.appId = appid1
	wx.appSecret = secret1
	if len(appid1) > 0 && len(secret1) > 0 {
		wx.tokenChan = make(chan accessToken)
		go createAccessToken(wx.tokenChan, appid1, secret1)
		//wx.ticketChan = make(chan jsApiTicket)
		//go createJsApiTicket(wx.tokenChan, wx.ticketChan)
	}
	return wx
}

func makeSignature(timestamp, nonce string) string {
	sl := []string{token, timestamp, nonce}
	sort.Strings(sl)
	s := sha1.New()
	io.WriteString(s, strings.Join(sl, ""))
	return fmt.Sprintf("%x", s.Sum(nil))
}

func validateUrl(w http.ResponseWriter, r *http.Request) bool {
	timestamp := strings.Join(r.Form["timestamp"], "")
	nonce := strings.Join(r.Form["nonce"], "")
	signatureGen := makeSignature(timestamp, nonce)

	signatureIn := strings.Join(r.Form["signature"], "")
	if signatureGen != signatureIn {
		return false
	}
	echostr := strings.Join(r.Form["echostr"], "")
	fmt.Fprintf(w, echostr)
	return true
}

func parseTextRequestBody(r *http.Request) *TextRequestBody {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	fmt.Println(string(body))
	requestBody := &TextRequestBody{}
	xml.Unmarshal(body, requestBody)
	return requestBody
}

func value2CDATA(v string) CDATAText {
	//return CDATAText{[]byte("<![CDATA[" + v + "]]>")}
	return CDATAText{"<![CDATA[" + v + "]]>"}
}

func makeTextResponseBody(fromUserName, toUserName, content string) ([]byte, error) {
	textResponseBody := &TextResponseBody{}
	textResponseBody.FromUserName = value2CDATA(fromUserName)
	textResponseBody.ToUserName = value2CDATA(toUserName)
	textResponseBody.MsgType = value2CDATA("text")
	textResponseBody.Content = value2CDATA(content)
	textResponseBody.CreateTime = time.Duration(time.Now().Unix())
	return xml.MarshalIndent(textResponseBody, " ", "  ")
}

func makeImgResponseBody(fromUserName, toUserName, mediaId string) ([]byte, error) {
	imgResponseBody := &ImgResponseBody{}
	imgResponseBody.FromUserName = value2CDATA(fromUserName)
	imgResponseBody.ToUserName = value2CDATA(toUserName)
	imgResponseBody.MsgType = value2CDATA("image")
	imgResponseBody.MediaId = value2CDATA(mediaId)
	imgResponseBody.CreateTime = time.Duration(time.Now().Unix())
	return xml.MarshalIndent(imgResponseBody, " ", "  ")
}

func authAccessToken(appid, secret string) (string, time.Duration) {
	resp, err := http.Get(weixinHost + "/token?grant_type=client_credential&appid=" + appid + "&secret=" + secret)
	if err != nil {
		log.Println("Get access token failed: ", err)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Read access token failed: ", err)
		} else {
			var res struct {
				AccessToken string `json:"access_token"`
				ExpiresIn   int64  `json:"expires_in"`
			}
			if err := json.Unmarshal(body, &res); err != nil {
				log.Println("Parse access token failed: ", err)
			} else {
				//log.Printf("AuthAccessToken token=%s expires_in=%d", res.AccessToken, res.ExpiresIn)
				return res.AccessToken, time.Duration(res.ExpiresIn * 1000 * 1000 * 1000)
			}
		}
	}
	return "", 0
}

func createAccessToken(c chan accessToken, appid string, secret string) {
	token := accessToken{"", time.Now()}
	c <- token
	for {
		if time.Since(token.expires).Seconds() >= 0 {
			var expires time.Duration
			token.token, expires = authAccessToken(appid, secret)
			token.expires = time.Now().Add(expires)
		}
		c <- token
	}
}

func uploadMedia(c chan accessToken, mediaType string, filename string, reader io.Reader) (string, error) {
	reqURL := weixinFileURL + "/upload?type=" + mediaType + "&access_token="
	for i := 0; i < retryMaxN; i++ {
		token := <-c
		if time.Since(token.expires).Seconds() < 0 {
			bodyBuf := &bytes.Buffer{}
			bodyWriter := multipart.NewWriter(bodyBuf)
			fileWriter, err := bodyWriter.CreateFormFile("filename", filename)
			if err != nil {
				return "", err
			}
			if _, err = io.Copy(fileWriter, reader); err != nil {
				return "", err
			}
			contentType := bodyWriter.FormDataContentType()
			bodyWriter.Close()
			r, err := http.Post(reqURL+token.token, contentType, bodyBuf)
			if err != nil {
				return "", err
			}
			defer r.Body.Close()
			reply, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return "", err
			}
			var result struct {
				response
				Type      string `json:"type"`
				MediaId   string `json:"media_id"`
				CreatedAt int64  `json:"created_at"`
			}
			err = json.Unmarshal(reply, &result)
			if err != nil {
				return "", err
			}
			// switch result.ErrorCode {
			// case 0:
			return result.MediaId, nil
			// case 42001: // access_token timeout and retry
			// 	continue
			// default:
			// 	return "", errors.New(fmt.Sprintf("WeiXin upload[%d]: %s", result.ErrorCode, result.ErrorMessage))
			// }
		}
	}
	return "", errors.New("WeiXin upload media too many times")
}

func procRequest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if !validateUrl(w, r) {
		log.Println("Wechat Service: this http request is not from Wechat platform!")
		return
	}

	if r.Method == "POST" {
		textRequestBody := parseTextRequestBody(r)
		if textRequestBody != nil {
			fmt.Printf("Wechat Service: Recv text msg [%s] from user [%s]!",
				textRequestBody.Content,
				textRequestBody.FromUserName)
			var content string
			if textRequestBody.Content == "0" {

				// content = "success"
				// responseTextBody, err := makeTextResponseBody(textRequestBody.ToUserName,
				// 	textRequestBody.FromUserName,
				// 	content)
				// if err != nil {
				// 	log.Println("Wechat Service: makeTextResponseBody error: ", err)
				// 	return
				// }
				// w.Header().Set("Content-Type", "text/xml")
				// fmt.Println(string(responseTextBody))
				// fmt.Fprintf(w, string(responseTextBody))

				filePath := haddle.WxLogin()
				mux := New(token, appID, appSecret)
				err := mux.PostText(textRequestBody.FromUserName, "success")
				if err != nil {
					log.Println(err)
				}

				MediaId1, _ := mux.UploadMediaFromFile("image", filePath)
				responseImgBody, err := makeImgResponseBody(textRequestBody.ToUserName,
					textRequestBody.FromUserName, MediaId1)
				if err != nil {
					log.Println("Wechat Service: makeTextResponseBody error: ", err)
					return
				}
				w.Header().Set("Content-Type", "text/xml")
				//fmt.Println(string(responseImgBody))
				//fmt.Fprintf(w, responseImgBody)
				w.Write([]byte(responseImgBody))
			} else {
				content = "输入的无效命令"
				responseTextBody, err := makeTextResponseBody(textRequestBody.ToUserName,
					textRequestBody.FromUserName,
					content)
				if err != nil {
					log.Println("Wechat Service: makeTextResponseBody error: ", err)
					return
				}
				w.Header().Set("Content-Type", "text/xml")
				fmt.Println(string(responseTextBody))
				fmt.Fprintf(w, string(responseTextBody))
			}

		}
	}
}

func main() {
	log.Println("Wechat Service: Start!")
	m := martini.Classic()
	m.Any("/", procRequest)
	//http.HandleFunc("/", procRequest)
	//err := http.ListenAndServe(":3001", nil)
	m.RunOnAddr(":3001")
	// if err != nil {
	// 	log.Fatal("Wechat Service: ListenAndServe failed, ", err)
	// }
	log.Println("Wechat Service: Stop!")
	m.Run()
}
