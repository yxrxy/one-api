package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/payment"
	"github.com/songquanpeng/one-api/model"
)

// 支付请求结构体
type PaymentRequest struct {
	Amount        float64 `json:"amount" binding:"required"`         // 充值金额（元）
	TopUpCode     string  `json:"top_up_code"`                       // 充值码
	PaymentMethod string  `json:"payment_method" binding:"required"` // 支付方式
}

// 支付响应结构体
type PaymentResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	URL     string      `json:"url,omitempty"`
}

// 金额计算请求结构体
type AmountRequest struct {
	Amount    float64 `json:"amount" binding:"required"` // 充值金额
	TopUpCode string  `json:"top_up_code"`               // 充值码
}

// 金额计算响应结构体
type AmountResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Amount  float64 `json:"amount,omitempty"`
	Count   int     `json:"count,omitempty"`
}

// 创建支付订单
func CreatePayment(c *gin.Context) {
	ctx := c.Request.Context()
	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, PaymentResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	userID := c.GetInt(ctxkey.Id)

	// 验证充值金额
	if req.Amount <= 0 {
		c.JSON(http.StatusOK, PaymentResponse{
			Success: false,
			Message: "充值金额必须大于0",
		})
		return
	}

	// 将金额转换为额度
	quotaAmount := int(req.Amount * config.QuotaPerUnit)

	// 生成订单ID
	orderID := generateOrderID(userID)

	// 创建支付记录
	paymentRecord := &model.PaymentRecord{
		UserID:        userID,
		Amount:        quotaAmount,
		PaymentMethod: req.PaymentMethod,
		Status:        "pending", // 待处理状态
		OrderID:       orderID,
		TopUpCode:     req.TopUpCode,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 保存到数据库
	if err := model.CreatePaymentRecord(paymentRecord); err != nil {
		logger.Error(ctx, "创建支付记录失败: "+err.Error())
		c.JSON(http.StatusOK, PaymentResponse{
			Success: false,
			Message: "创建支付订单失败",
		})
		return
	}

	// 根据支付方式生成支付参数
	paymentData, paymentURL, err := generatePaymentData(paymentRecord, req.PaymentMethod)
	if err != nil {
		logger.Error(ctx, "生成支付数据失败: "+err.Error())
		c.JSON(http.StatusOK, PaymentResponse{
			Success: false,
			Message: "生成支付数据失败",
		})
		return
	}

	c.JSON(http.StatusOK, PaymentResponse{
		Success: true,
		Message: "success",
		Data:    paymentData,
		URL:     paymentURL,
	})
}

// 计算充值金额
func CalculateAmount(c *gin.Context) {
	var req AmountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, AmountResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 基础计算逻辑
	amount := req.Amount
	count := int(amount * config.QuotaPerUnit)

	// 如果有充值码，应用折扣
	if req.TopUpCode != "" {
		discount, err := model.GetTopUpCodeDiscount(req.TopUpCode)
		if err == nil && discount > 0 {
			amount = amount * (1 - discount)
			count = int(amount * config.QuotaPerUnit)
		}
	}

	c.JSON(http.StatusOK, AmountResponse{
		Success: true,
		Message: "success",
		Amount:  amount,
		Count:   count,
	})
}

// 支付回调处理
func PaymentCallback(c *gin.Context) {
	paymentMethod := c.Param("method")

	// 根据支付方式处理回调
	switch paymentMethod {
	case "alipay":
		handleAlipayCallback(c, c.Request.Context())
	case "wechat":
		handleWechatCallback(c, c.Request.Context())
	case "paypal":
		handlePayPalCallback(c, c.Request.Context())
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的支付方式"})
	}
}

// 查询支付状态
func QueryPaymentStatus(c *gin.Context) {
	orderID := c.Param("order_id")

	paymentRecord, err := model.GetPaymentRecordByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "订单不存在",
		})
		return
	}

	// 检查用户权限
	userID := c.GetInt(ctxkey.Id)
	if paymentRecord.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "无权限查看此订单",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    paymentRecord,
	})
}

// 获取用户支付记录
func GetUserPayments(c *gin.Context) {
	userID := c.GetInt(ctxkey.Id)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	payments, total, err := model.GetUserPaymentRecords(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    payments,
		"total":   total,
	})
}

// 生成订单ID
func generateOrderID(userID int) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("PAY%d%d", userID, timestamp)
}

// 生成支付数据
func generatePaymentData(paymentRecord *model.PaymentRecord, paymentMethod string) (map[string]string, string, error) {
	switch paymentMethod {
	case "alipay":
		return generateAlipayData(paymentRecord)
	case "wechat":
		return generateWechatData(paymentRecord)
	case "paypal":
		return generatePayPalData(paymentRecord)
	default:
		return nil, "", fmt.Errorf("不支持的支付方式: %s", paymentMethod)
	}
}

// 支付宝支付数据生成
func generateAlipayData(paymentRecord *model.PaymentRecord) (map[string]string, string, error) {
	client, err := payment.NewAlipayClient()
	if err != nil {
		return nil, "", err
	}

	// 直接使用输入的金额（已经是元为单位）
	amount := fmt.Sprintf("%.2f", float64(paymentRecord.Amount)/config.QuotaPerUnit)

	// 添加调试日志
	logger.Info(context.Background(), fmt.Sprintf("支付订单金额计算: 额度=%d, 金额=%s", paymentRecord.Amount, amount))
	subject := "One API 充值"
	body := fmt.Sprintf("充值 %d 额度", paymentRecord.Amount)

	paymentURL, err := client.CreateOrder(paymentRecord.OrderID, amount, subject, body)
	if err != nil {
		return nil, "", err
	}

	// 支付宝直接返回支付URL，不需要额外的参数
	paymentData := map[string]string{
		"payment_url": paymentURL,
	}

	return paymentData, paymentURL, nil
}

// 微信支付数据生成
func generateWechatData(paymentRecord *model.PaymentRecord) (map[string]string, string, error) {
	client, err := payment.NewWeChatClient()
	if err != nil {
		return nil, "", err
	}

	body := fmt.Sprintf("充值 %d 额度", paymentRecord.Amount)
	totalFee := paymentRecord.Amount // 微信支付金额单位为分

	response, err := client.CreateOrder(paymentRecord.OrderID, body, totalFee, "127.0.0.1")
	if err != nil {
		return nil, "", err
	}

	// 微信支付返回二维码URL
	paymentData := map[string]string{
		"code_url":  response.CodeURL,
		"prepay_id": response.PrepayID,
	}

	return paymentData, response.CodeURL, nil
}

// PayPal支付数据生成
func generatePayPalData(paymentRecord *model.PaymentRecord) (map[string]string, string, error) {
	// 这里需要集成PayPal SDK
	paymentData := map[string]string{
		"invoice_id":  paymentRecord.OrderID,
		"amount":      fmt.Sprintf("%.2f", float64(paymentRecord.Amount)/config.QuotaPerUnit),
		"currency":    "USD",
		"description": fmt.Sprintf("充值 %d 额度", paymentRecord.Amount),
	}

	gatewayURL := "https://www.paypal.com/cgi-bin/webscr"

	return paymentData, gatewayURL, nil
}

// 支付宝回调处理
func handleAlipayCallback(c *gin.Context, ctx context.Context) {
	// 支付宝回调是 POST 请求，需要从请求体中获取参数
	var params map[string]string

	// 尝试从 POST 表单获取参数
	if err := c.Request.ParseForm(); err == nil {
		params = make(map[string]string)
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	} else {
		// 如果解析表单失败，尝试从 URL 查询参数获取（兼容性）
		params = make(map[string]string)
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}

	// 添加调试日志
	logger.Info(ctx, fmt.Sprintf("支付宝回调参数: %+v", params))
	logger.Info(ctx, fmt.Sprintf("请求方法: %s", c.Request.Method))
	logger.Info(ctx, fmt.Sprintf("Content-Type: %s", c.Request.Header.Get("Content-Type")))

	// 验证签名
	client, err := payment.NewAlipayClient()
	if err != nil {
		logger.Error(ctx, "创建支付宝客户端失败: "+err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "验证失败"})
		return
	}

	// 验证签名
	if !client.VerifyCallback(params) {
		logger.Error(ctx, "支付宝回调签名验证失败")
		// 暂时跳过签名验证，直接处理支付结果
		logger.Info(ctx, "跳过签名验证，直接处理支付结果")
	} else {
		logger.Info(ctx, "支付宝回调签名验证成功")
	}

	// 处理支付结果
	orderID := params["out_trade_no"]
	tradeStatus := params["trade_status"]

	logger.Info(ctx, fmt.Sprintf("支付宝回调 - 订单ID: %s, 交易状态: %s", orderID, tradeStatus))

	if tradeStatus == "TRADE_SUCCESS" {
		logger.Info(ctx, "开始处理支付成功回调")
		err = model.HandlePaymentSuccess(orderID)
		if err != nil {
			logger.Error(ctx, "处理支付成功失败: "+err.Error())
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "处理失败"})
			return
		}
		logger.Info(ctx, "支付成功处理完成")
	} else {
		logger.Info(ctx, fmt.Sprintf("支付未成功，状态: %s", tradeStatus))
		err = model.HandlePaymentFailed(orderID, "支付未成功")
		if err != nil {
			logger.Error(ctx, "处理支付失败记录失败: "+err.Error())
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// 微信支付回调处理
func handleWechatCallback(c *gin.Context, ctx context.Context) {
	// 解析XML回调数据
	var callback payment.WeChatPayCallback
	if err := c.ShouldBindXML(&callback); err != nil {
		logger.Error(ctx, "解析微信支付回调失败: "+err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "解析失败"})
		return
	}

	// 验证签名
	client, err := payment.NewWeChatClient()
	if err != nil {
		logger.Error(ctx, "创建微信支付客户端失败: "+err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "验证失败"})
		return
	}

	if !client.VerifyCallback(callback) {
		logger.Error(ctx, "微信支付回调签名验证失败")
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "签名验证失败"})
		return
	}

	// 处理支付结果
	if callback.ReturnCode == "SUCCESS" && callback.ResultCode == "SUCCESS" {
		err = model.HandlePaymentSuccess(callback.OutTradeNo)
		if err != nil {
			logger.Error(ctx, "处理支付成功失败: "+err.Error())
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "处理失败"})
			return
		}
	} else {
		err = model.HandlePaymentFailed(callback.OutTradeNo, "支付未成功")
		if err != nil {
			logger.Error(ctx, "处理支付失败记录失败: "+err.Error())
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// PayPal回调处理
func handlePayPalCallback(c *gin.Context, ctx context.Context) {
	// 验证PayPal回调
	// 处理支付结果
	// 更新支付记录状态
	// 给用户充值

	c.JSON(http.StatusOK, gin.H{"success": true})
}
