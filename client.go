package weixin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/levigross/grequests"
	"github.com/tidwall/gjson"
)

const (
	ACCESSTOKEN_API_URL = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	API_URI             = "https://qyapi.weixin.qq.com/cgi-bin/"
)

var (
	// WeixinErr 微信错误信息码
	WeixinErr = func(errcode int64, errmsg string) error {
		return fmt.Errorf("weixin return error, errcode: %d, errmsg: %s", errcode, errmsg)
	}
)

// Client 微信
type Client struct {
	CorpID     string
	CorpSecret string
	accessInfo accessInfo
}

type accessInfo struct {
	token   string
	expired time.Time
}

func NewClient(corpId string, CorpSecret string) *Client {
	return &Client{
		CorpID:     corpId,
		CorpSecret: CorpSecret,
		accessInfo: accessInfo{
			token:   "",
			expired: time.Now(),
		},
	}
}

// GetAccessToken 获取AccessToken
// corpid 每个企业都拥有唯一的corpid，获取此信息可在管理后台“我的企业”－“企业信息”下查看（需要有管理员权限）
// corpsecret 每个应用有独立的secret，所以每个应用的access_token应该分开来获取 在管理后台->“企业应用”->点进应用
func (c *Client) GetAccessToken() (string, error) {
	now := time.Now()
	if c.accessInfo.token != "" && c.accessInfo.expired.After(now) {
		return c.accessInfo.token, nil
	}

	o := &grequests.RequestOptions{
		Params: map[string]string{
			"corpid":     c.CorpID,
			"corpsecret": c.CorpSecret,
		},
	}

	resp, err := grequests.Get(ACCESSTOKEN_API_URL, o)
	if err != nil {
		return "", err
	}

	respJSON := resp.String()
	errcode := gjson.Get(respJSON, "errcode")
	if errcode.Int() == 0 {
		token := gjson.Get(respJSON, "access_token")
		expiresIn := gjson.Get(respJSON, "expires_in")
		c.accessInfo.token = token.String()
		c.accessInfo.expired = now.Add(time.Duration(expiresIn.Int()-5) * time.Second)
		return c.accessInfo.token, nil
	}
	return "", WeixinErr(errcode.Int(), gjson.Get(respJSON, "errmsg").String())
}

/*
* 调用api,发送消息
 */
func (c *Client) Send(apiUri string, requestType string, body []byte) (*grequests.Response, error) {
	var resp *grequests.Response
	var err error
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}
	o := &grequests.RequestOptions{
		Params: map[string]string{
			"access_token": accessToken,
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	if len(body) > 0 {
		o.JSON = body
	}
	apiUrl := API_URI + apiUri
	switch requestType {
	case "GET":
		resp, err = grequests.Get(apiUrl, o)
	case "POST":
		resp, err = grequests.Post(apiUrl, o)
	}
	if err != nil {
		return nil, err
	}
	respJSON := resp.String()
	errcode := gjson.Get(respJSON, "errcode").Int()
	if errcode == 0 {
		return resp, nil
	}
	return nil, WeixinErr(errcode, gjson.Get(respJSON, "errmsg").String())
}

// SendMessage 发送消息
func (c *Client) SendMessage(msg *Message) (bool, error) {
	reqJSON, err := json.Marshal(msg)
	if err != nil {
		return false, err
	}
	_, err = c.Send("message/send", "POST", reqJSON)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}
