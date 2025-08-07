package payment

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common/config"
)

// 支付宝支付客户端
type AlipayClient struct {
	AppID      string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Gateway    string
}

// 支付宝请求参数
type AlipayRequest struct {
	AppID      string `json:"app_id"`
	Method     string `json:"method"`
	Format     string `json:"format"`
	Charset    string `json:"charset"`
	SignType   string `json:"sign_type"`
	Sign       string `json:"sign"`
	Timestamp  string `json:"timestamp"`
	Version    string `json:"version"`
	NotifyURL  string `json:"notify_url"`
	BizContent string `json:"biz_content"`
}

// 支付宝业务参数
type AlipayBizContent struct {
	OutTradeNo  string `json:"out_trade_no"`
	TotalAmount string `json:"total_amount"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	ProductCode string `json:"product_code"`
}

// 创建支付宝客户端
func NewAlipayClient() (*AlipayClient, error) {
	if config.AlipayAppID == "" || config.AlipayPrivateKey == "" {
		return nil, fmt.Errorf("支付宝配置不完整")
	}

	// 解析私钥
	privateKey, err := parsePrivateKey(config.AlipayPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %v", err)
	}

	// 解析公钥
	publicKey, err := parsePublicKey(config.AlipayPublicKey)
	if err != nil {
		return nil, fmt.Errorf("解析公钥失败: %v", err)
	}

	return &AlipayClient{
		AppID:      config.AlipayAppID,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Gateway:    "https://openapi.alipay.com/gateway.do",
	}, nil
}

// 创建支付订单
func (c *AlipayClient) CreateOrder(orderID, amount, subject, body string) (string, error) {
	bizContent := AlipayBizContent{
		OutTradeNo:  orderID,
		TotalAmount: amount,
		Subject:     subject,
		Body:        body,
		ProductCode: "FAST_INSTANT_TRADE_PAY",
	}

	bizContentBytes, err := json.Marshal(bizContent)
	if err != nil {
		return "", err
	}

	params := map[string]string{
		"app_id":      c.AppID,
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  config.ServerAddress + "/api/payment/callback/alipay",
		"biz_content": string(bizContentBytes),
	}

	// 生成签名
	sign, err := c.sign(params)
	if err != nil {
		return "", err
	}
	params["sign"] = sign

	// 构建请求URL
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	return c.Gateway + "?" + values.Encode(), nil
}

// 验证回调签名
func (c *AlipayClient) VerifyCallback(params map[string]string) bool {
	sign := params["sign"]
	delete(params, "sign")
	delete(params, "sign_type")

	// 构建签名字符串
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var signStr strings.Builder
	for _, k := range keys {
		if params[k] != "" {
			signStr.WriteString(k + "=" + params[k] + "&")
		}
	}
	signStrStr := strings.TrimSuffix(signStr.String(), "&")

	// 验证签名
	hashed := sha256.Sum256([]byte(signStrStr))
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return false
	}

	err = rsa.VerifyPKCS1v15(c.PublicKey, crypto.SHA256, hashed[:], signBytes)
	return err == nil
}

// 生成签名
func (c *AlipayClient) sign(params map[string]string) (string, error) {
	// 构建签名字符串
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var signStr strings.Builder
	for _, k := range keys {
		if params[k] != "" {
			signStr.WriteString(k + "=" + params[k] + "&")
		}
	}
	signStrStr := strings.TrimSuffix(signStr.String(), "&")

	// 生成签名
	hashed := sha256.Sum256([]byte(signStrStr))
	signBytes, err := rsa.SignPKCS1v15(rand.Reader, c.PrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signBytes), nil
}

// 解析私钥
func parsePrivateKey(privateKeyStr string) (*rsa.PrivateKey, error) {
	// 首先尝试解析 PEM 格式
	block, _ := pem.Decode([]byte(privateKeyStr))
	if block != nil {
		// 尝试 PKCS1 格式
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err == nil {
			return privateKey, nil
		}

		// 尝试 PKCS8 格式
		privateKeyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: 不是有效的 PKCS1 或 PKCS8 格式")
		}

		privateKey, ok := privateKeyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("不是 RSA 私钥")
		}

		return privateKey, nil
	}

	// 如果不是 PEM 格式，尝试解析 Base64 格式
	keyBytes, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: 不是有效的 PEM 或 Base64 格式")
	}

	// 尝试 PKCS1 格式
	privateKey, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err == nil {
		return privateKey, nil
	}

	// 尝试 PKCS8 格式
	privateKeyInterface, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: 不是有效的 PKCS1 或 PKCS8 格式")
	}

	privateKey, ok := privateKeyInterface.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("不是 RSA 私钥")
	}

	return privateKey, nil
}

// 解析公钥
func parsePublicKey(publicKeyStr string) (*rsa.PublicKey, error) {
	// 首先尝试解析 PEM 格式
	block, _ := pem.Decode([]byte(publicKeyStr))
	if block != nil {
		publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}

		rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("不是RSA公钥")
		}

		return rsaPublicKey, nil
	}

	// 如果不是 PEM 格式，尝试解析 Base64 格式
	keyBytes, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		return nil, fmt.Errorf("解析公钥失败: 不是有效的 PEM 或 Base64 格式")
	}

	publicKey, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return nil, err
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("不是RSA公钥")
	}

	return rsaPublicKey, nil
}
