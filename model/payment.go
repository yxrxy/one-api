package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/songquanpeng/one-api/common"
	"gorm.io/gorm"
)

// 支付记录表
type PaymentRecord struct {
	ID            int       `json:"id" gorm:"primaryKey"`
	UserID        int       `json:"user_id" gorm:"index"`
	Amount        int       `json:"amount"`                                              // 充值数量
	PaymentMethod string    `json:"payment_method"`                                      // 支付方式
	Status        string    `json:"status"`                                              // 支付状态: pending, processing, success, failed
	OrderID       string    `json:"order_id" gorm:"uniqueIndex:idx_order_id,length:100"` // 订单ID
	TopUpCode     string    `json:"top_up_code"`                                         // 充值码
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// 充值码折扣表
type TopUpCode struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	Code      string    `json:"code" gorm:"uniqueIndex:idx_code,length:100"` // 充值码
	Discount  float64   `json:"discount"`                                    // 折扣比例 (0-1)
	Status    string    `json:"status"`                                      // 状态: enabled, disabled
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 创建支付记录
func CreatePaymentRecord(paymentRecord *PaymentRecord) error {
	return DB.Create(paymentRecord).Error
}

// 根据订单ID获取支付记录
func GetPaymentRecordByOrderID(orderID string) (*PaymentRecord, error) {
	var paymentRecord PaymentRecord
	err := DB.Where("order_id = ?", orderID).First(&paymentRecord).Error
	if err != nil {
		return nil, err
	}
	return &paymentRecord, nil
}

// 更新支付记录状态
func UpdatePaymentRecordStatus(orderID string, status string) error {
	return DB.Model(&PaymentRecord{}).Where("order_id = ?", orderID).Update("status", status).Error
}

// 获取充值码折扣
func GetTopUpCodeDiscount(code string) (float64, error) {
	var topUpCode TopUpCode
	err := DB.Where("code = ? AND status = ?", code, "enabled").First(&topUpCode).Error
	if err != nil {
		return 0, err
	}
	return topUpCode.Discount, nil
}

// 创建充值码
func CreateTopUpCode(topUpCode *TopUpCode) error {
	return DB.Create(topUpCode).Error
}

// 获取所有支付记录
func GetAllPaymentRecords(page, pageSize int) ([]PaymentRecord, int64, error) {
	var paymentRecords []PaymentRecord
	var total int64

	err := DB.Model(&PaymentRecord{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = DB.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&paymentRecords).Error
	if err != nil {
		return nil, 0, err
	}

	return paymentRecords, total, nil
}

// 获取用户支付记录
func GetUserPaymentRecords(userID int, page, pageSize int) ([]PaymentRecord, int64, error) {
	var paymentRecords []PaymentRecord
	var total int64

	err := DB.Model(&PaymentRecord{}).Where("user_id = ?", userID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = DB.Where("user_id = ?", userID).Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&paymentRecords).Error
	if err != nil {
		return nil, 0, err
	}

	return paymentRecords, total, nil
}

// 处理支付成功
func HandlePaymentSuccess(orderID string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// 获取支付记录
		var paymentRecord PaymentRecord
		err := tx.Where("order_id = ?", orderID).First(&paymentRecord).Error
		if err != nil {
			return err
		}

		// 检查是否已经处理过
		if paymentRecord.Status == "success" {
			return errors.New("订单已处理")
		}

		// 检查是否正在处理中
		if paymentRecord.Status == "processing" {
			return errors.New("订单正在处理中")
		}

		// 先更新为处理中状态，防止重复处理
		err = tx.Model(&paymentRecord).Update("status", "processing").Error
		if err != nil {
			return err
		}

		// 给用户充值
		err = IncreaseUserQuota(paymentRecord.UserID, int64(paymentRecord.Amount))
		if err != nil {
			// 充值失败，回滚状态
			tx.Model(&paymentRecord).Update("status", "pending")
			return err
		}

		// 更新为成功状态
		err = tx.Model(&paymentRecord).Update("status", "success").Error
		if err != nil {
			return err
		}

		// 记录充值日志
		RecordTopupLog(context.TODO(), paymentRecord.UserID,
			fmt.Sprintf("在线支付充值 %s", common.LogQuota(int64(paymentRecord.Amount))), int(paymentRecord.Amount))

		return nil
	})
}

// 处理支付失败
func HandlePaymentFailed(orderID string, reason string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// 获取支付记录
		var paymentRecord PaymentRecord
		err := tx.Where("order_id = ?", orderID).First(&paymentRecord).Error
		if err != nil {
			return err
		}

		// 检查是否已经处理过
		if paymentRecord.Status == "success" || paymentRecord.Status == "failed" {
			return errors.New("订单已处理")
		}

		// 更新支付状态
		err = tx.Model(&paymentRecord).Update("status", "failed").Error
		if err != nil {
			return err
		}

		// 记录失败日志
		RecordLog(context.TODO(), paymentRecord.UserID, LogTypeTopup,
			fmt.Sprintf("支付失败: %s", reason))

		return nil
	})
}
