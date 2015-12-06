package haddle

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	//"strings"
)

var gCurCookies []*http.Cookie
var gCurCookieJar *cookiejar.Jar

func initAll() {
	fmt.Println("22")
	gCurCookies = nil
	//var err error;
	gCurCookieJar, _ = cookiejar.New(nil)
}

const (
	verify_code_url string = "http://jwc.sut.edu.cn/ACTIONVALIDATERANDOMPICTURE.APPPROCESS"
	getUrl          string = "http://jwc.sut.edu.cn/ACTIONQUERYSTUDENTPIC.APPPROCESS?ByStudentNO=null"
	login_url       string = "http://jwc.sut.edu.cn/ACTIONLOGON.APPPROCESS?mode=3"
	post_login_url  string = "http://jwc.sut.edu.cn/ACTIONLOGON.APPPROCESS?mode=4"
	uname           string = "130405212"
	pwd             string = "930904"
)

func login() string {
	httpClient := &http.Client{
		CheckRedirect: nil,
		Jar:           gCurCookieJar,
	}
	fmt.Println("32")

	req, _ := http.NewRequest("GET", login_url, nil)
	res, _ := httpClient.Do(req)

	req.URL, _ = url.Parse(verify_code_url)
	//var temp_cookies = res.Cookies()
	fmt.Println("33")
	for _, v := range res.Cookies() {
		req.AddCookie(v)
	}

	// 获取验证码
	//var verify_code string
	//for {
	res, _ = httpClient.Do(req)
	file, _ := os.Create("verify.gif")
	io.Copy(file, res.Body)

	fmt.Println("请查看verify.gif， 然后输入验证码， 看不清输入0重新获取验证码")
	//fmt.Scanf("%s", &verify_code)
	// if verify_code != "0" {
	// 	break
	// }
	//filePath, _ := os.Getwd()

	//fi := strings.Split(filePath, "/")
	//filePath = strings.Join(fi[0:len(fi)-2], "/")

	res.Body.Close()
	//return filePath + "/verify.gif"
	return "go/src/wx/verify.gif"
	//}
	// v := url.Values{}
	// v.Add("WebUserNO", uname)
	// v.Add("Password", pwd)
	// v.Add("Agnomen", verify_code)
	// postURL, _ := url.Parse(post_login_url)
	// httpClient.Jar.SetCookies(postURL, temp_cookies)
	// res, _ = httpClient.PostForm(post_login_url, v)
	// gCurCookies = res.Cookies()
}

func getBody() string {
	url, _ := url.Parse(getUrl)
	gCurCookieJar.SetCookies(url, gCurCookies)
	httpClient := &http.Client{
		CheckRedirect: nil,
		Jar:           gCurCookieJar,
	}
	// header := make(http.Header)
	// header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Get(getUrl)
	if err != nil {
		fmt.Println("login出错:", err.Error())
	}
	file, _ := os.Create("phone.gif")
	io.Copy(file, resp.Body)
	body, err := ioutil.ReadAll(resp.Body)

	return string(body)
}

func WxLogin() string {
	fmt.Println("11")
	initAll()
	filePath := login()
	return filePath
	//data := getBody()
	//fmt.Println(data)
}
