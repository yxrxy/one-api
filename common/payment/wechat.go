package payment

import (
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common/config"
)

// 微信支付客户端
type WeChatClient struct {
	AppID   string
	MchID   string
	Key     string
	Gateway string
}

// 微信支付请求参数
type WeChatPayRequest struct {
	AppID          string `xml:"appid"`
	MchID          string `xml:"mch_id"`
	NonceStr       string `xml:"nonce_str"`
	Sign           string `xml:"sign"`
	Body           string `xml:"body"`
	OutTradeNo     string `xml:"out_trade_no"`
	TotalFee       int    `xml:"total_fee"`
	SpbillCreateIP string `xml:"spbill_create_ip"`
	NotifyURL      string `xml:"notify_url"`
	TradeType      string `xml:"trade_type"`
}

// 微信支付响应
type WeChatPayResponse struct {
	ReturnCode string `xml:"return_code"`
	ReturnMsg  string `xml:"return_msg"`
	ResultCode string `xml:"result_code"`
	PrepayID   string `xml:"prepay_id"`
	CodeURL    string `xml:"code_url"`
}

// 微信支付回调
type WeChatPayCallback struct {
	ReturnCode string `xml:"return_code"`
	ResultCode string `xml:"result_code"`
	OutTradeNo string `xml:"out_trade_no"`
	TotalFee   int    `xml:"total_fee"`
	Sign       string `xml:"sign"`
}

// 创建微信支付客户端
func NewWeChatClient() (*WeChatClient, error) {
	if config.WeChatAppID == "" || config.WeChatMchID == "" || config.WeChatKey == "" {
		return nil, fmt.Errorf("微信支付配置不完整")
	}

	return &WeChatClient{
		AppID:   config.WeChatAppID,
		MchID:   config.WeChatMchID,
		Key:     config.WeChatKey,
		Gateway: "https://api.mch.weixin.qq.com/pay/unifiedorder",
	}, nil
}

// 创建支付订单
func (c *WeChatClient) CreateOrder(orderID, body string, totalFee int, clientIP string) (*WeChatPayResponse, error) {
	request := WeChatPayRequest{
		AppID:          c.AppID,
		MchID:          c.MchID,
		NonceStr:       generateNonceStr(),
		Body:           body,
		OutTradeNo:     orderID,
		TotalFee:       totalFee,
		SpbillCreateIP: clientIP,
		NotifyURL:      config.ServerAddress + "/api/payment/callback/wechat",
		TradeType:      "NATIVE", // 二维码支付
	}

	// 生成签名
	request.Sign = c.sign(request)

	// 发送请求
	xmlData, err := xml.Marshal(request)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.Gateway, "application/xml", strings.NewReader(string(xmlData)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response WeChatPayResponse
	err = xml.Unmarshal(bodyBytes, &response)
	if err != nil {
		return nil, err
	}

	if response.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("微信支付请求失败: %s", response.ReturnMsg)
	}

	if response.ResultCode != "SUCCESS" {
		return nil, fmt.Errorf("微信支付业务失败")
	}

	return &response, nil
}

// 验证回调签名
func (c *WeChatClient) VerifyCallback(callback WeChatPayCallback) bool {
	// 构建签名字符串
	params := map[string]string{
		"return_code":  callback.ReturnCode,
		"result_code":  callback.ResultCode,
		"out_trade_no": callback.OutTradeNo,
		"total_fee":    fmt.Sprintf("%d", callback.TotalFee),
	}

	signStr := c.buildSignString(params)
	expectedSign := strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(signStr))))

	return expectedSign == callback.Sign
}

// 生成签名
func (c *WeChatClient) sign(request WeChatPayRequest) string {
	params := map[string]string{
		"appid":            request.AppID,
		"mch_id":           request.MchID,
		"nonce_str":        request.NonceStr,
		"body":             request.Body,
		"out_trade_no":     request.OutTradeNo,
		"total_fee":        fmt.Sprintf("%d", request.TotalFee),
		"spbill_create_ip": request.SpbillCreateIP,
		"notify_url":       request.NotifyURL,
		"trade_type":       request.TradeType,
	}

	signStr := c.buildSignString(params)
	return strings.ToUpper(fmt.Sprintf("%x", md5.Sum([]byte(signStr))))
}

// 构建签名字符串
func (c *WeChatClient) buildSignString(params map[string]string) string {
	var keys []string
	for k := range params {
		if params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var signStr strings.Builder
	for _, k := range keys {
		signStr.WriteString(k + "=" + params[k] + "&")
	}
	signStr.WriteString("key=" + c.Key)

	return signStr.String()
}

// 生成随机字符串
func generateNonceStr() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
