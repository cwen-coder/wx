package main

import (
	//"github.com/go-martini/martini"
	"fmt"
	"github.com/wizjin/weixin"
	"log"
	"net/http"
)

const (
	token     = "wxcwenyin"
	appID     = "wx1abd680617d17f5f"
	appSecret = "7aa1af925a4483b126be488502c5e7db"
)

func Echo(w weixin.ResponseWriter, r *weixin.Request) {
	txt := r.Content // 获取用户发送的消息
	w.ReplyText(txt) // 回复一条文本消息
	//w.PostText("Post:" + txt) // 发送一条文本消息
	ShortURL(w)
}

// 关注事件的处理函数
func Subscribe(w weixin.ResponseWriter, r *weixin.Request) {
	w.ReplyText("欢迎关注") // 有新人关注，返回欢迎消息
}

func main() {
	log.Println("Wechat Service: Start!")
	mux := weixin.New(token, appID, appSecret)
	//m := martini.Classic()
	mux.HandleFunc(weixin.MsgTypeText, Echo)
	// 注册关注事件的处理函数
	mux.HandleFunc(weixin.MsgTypeEventSubscribe, Subscribe)
	http.Handle("/", mux) // 注册接收微信服务器数据的接口URI
	//http.HandleFunc("/", procRequest)
	//err := http.ListenAndServe(":3001", nil)
	//m.RunOnAddr(":3001")
	// if err != nil {
	// 	log.Fatal("Wechat Service: ListenAndServe failed, ", err)
	// }
	http.ListenAndServe(":3001", nil) // 启动接收微信数据服务器
	log.Println("Wechat Service: Stop!")
	//m.Run()
}
