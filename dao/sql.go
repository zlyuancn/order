package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/crc32"
	"time"

	"github.com/didi/gendry/builder"
	"github.com/spf13/cast"
	"github.com/zly-app/zapp/logger"
	"go.uber.org/zap"

	"github.com/zlyuancn/order/client"
	"github.com/zlyuancn/order/conf"
	"github.com/zlyuancn/order/order_model"
)

var (
	// Dao 对外暴露实例
	Dao = func(uid string) RPC {
		shardID := crc32.ChecksumIEEE([]byte(uid)) % conf.Conf.TableShardNums
		return &impl{
			tabName: TableName + cast.ToString(shardID),
			uid:     uid,
		}
	}
)

type impl struct {
	uid     string
	tabName string
}

func (i *impl) CreateOneModel(ctx context.Context, v *Model) (int64, error) {
	if v == nil {
		return 0, errors.New("CreateOneModel v is empty")
	}
	var data []map[string]interface{}
	data = append(data, map[string]interface{}{
		"oid":      v.OrderID,
		"o_type":   v.OrderType,
		"o_status": v.OrderStatus,

		"pay_type":      v.PayType,
		"pay_status":    v.PayStatus,
		"pay_amount":    v.PayAmount,
		"third_pay_oid": v.ThirdPayOrderID,

		"uid":    v.Uid,
		"extend": v.Extend,
		"remark": v.Remark,
	})
	cond, vals, err := builder.BuildInsert(i.tabName, data)
	if err != nil {
		logger.Log.Error(ctx, "order CreateOneModel BuildSelect err",
			zap.Any("data", data),
			zap.Error(err),
		)
		return 0, err
	}

	result, err := client.SqlxClient.Exec(ctx, cond, vals...)
	if err != nil {
		logger.Log.Error(ctx, "order CreateOneModel err",
			zap.String("cond", cond),
			zap.Any("vals", vals),
			zap.Error(err),
		)
		return 0, err
	}
	return result.LastInsertId()
}

var getOneSelectField = []string{
	"id",
	"o_type",
	"o_status",

	"pay_type",
	"pay_status",
	"pay_amount",
	"third_pay_oid",

	"extend",
	"remark",
}

func (i *impl) GetOne(ctx context.Context, orderID string) (*Model, error) {
	where := map[string]interface{}{
		"oid":    orderID,
		"uid":    i.uid,
		"_limit": []uint{1},
	}
	cond, vals, err := builder.BuildSelect(i.tabName, where, getOneSelectField)
	if nil != err {
		logger.Log.Error(ctx, "order CreateOneModel BuildSelect err",
			zap.Any("select", getOneSelectField),
			zap.Any("where", where),
			zap.Error(err),
		)
		return nil, err
	}
	var ret = &Model{}
	err = client.SqlxClient.QueryToStruct(ctx, ret, cond, vals...)
	if nil != err {
		if err != sql.ErrNoRows {
			logger.Log.Error(ctx, "order GetOne err",
				zap.String("cond", cond),
				zap.Any("vals", vals),
				zap.Error(err),
			)
		}
		return nil, err
	}
	ret.OrderID = orderID
	ret.Uid = i.uid
	return ret, err
}

func (i *impl) UpdateOrderStatus(ctx context.Context, orderID string, extend string, status order_model.OrderStatus,
	remark string) error {
	cond := `update ` + i.tabName + ` set o_status=?`
	vals := []interface{}{status}

	if extend != "" {
		cond += `, extend=?`
		vals = append(vals, extend)
	}

	cond += `, remark=?, update_nums=update_nums + 1, utime=now() where oid=? limit 1;`
	vals = append(vals, remark, orderID)

	result, err := client.SqlxClient.Exec(ctx, cond, vals...)
	if err != nil {
		logger.Log.Error(ctx, "order updateOrderStatus err",
			zap.String("cond", cond),
			zap.Any("vals", vals),
			zap.Error(err),
		)
		return err
	}

	nums, err := result.RowsAffected()
	if err != nil {
		logger.Log.Error(ctx, "order updateOrderStatus get RowsAffected err",
			zap.String("cond", cond),
			zap.Any("vals", vals),
			zap.Error(err),
		)
		return err
	}
	if nums != 1 {
		logger.Log.Error(ctx, "order updateOrderStatus nums != 1",
			zap.String("cond", cond),
			zap.Any("vals", vals),
			zap.Int64("nums", nums),
		)
		return fmt.Errorf("order updateOrderStatus nums!=1 is %v", nums)
	}
	return nil
}

func (i *impl) SetPayStatus(ctx context.Context, orderID, thirdPayOid string, payStatus byte, remark string) error {
	cond := `update ` + i.tabName + ` set pay_status=?, remark=?, update_nums=update_nums + 1, utime=now() where `
	vals := []interface{}{payStatus, remark}
	if orderID != "" {
		cond += `oid=? limit 1;`
		vals = append(vals, orderID)
	} else if thirdPayOid != "" {
		cond += `third_pay_oid=? limit 1;`
		vals = append(vals, thirdPayOid)
	} else {
		logger.Log.Error(ctx, "order SetPayStatus args err. orderID and thirdPayOid is empty")
		return errors.New("order SetPayStatus args err. orderID and thirdPayOid is empty")
	}

	result, err := client.SqlxClient.Exec(ctx, cond, vals...)
	if err != nil {
		logger.Log.Error(ctx, "order SetPayStatus err",
			zap.String("cond", cond),
			zap.Any("vals", vals),
			zap.Error(err),
		)
		return err
	}

	nums, err := result.RowsAffected()
	if err != nil {
		logger.Log.Error(ctx, "order SetPayStatus get RowsAffected err",
			zap.String("cond", cond),
			zap.Any("vals", vals),
			zap.Error(err),
		)
		return err
	}
	if nums != 1 {
		logger.Log.Error(ctx, "order SetPayStatus nums != 1",
			zap.String("cond", cond),
			zap.Any("vals", vals),
			zap.Int64("nums", nums),
		)
		return fmt.Errorf("order SetPayStatus nums!=1 is %v", nums)
	}
	return nil
}

const TableName = "order_"

// RPC 接口
type RPC interface {
	CreateOneModel(ctx context.Context, v *Model) (int64, error)
	GetOne(ctx context.Context, orderID string) (*Model, error)

	/*更新订单状态. 在绝大部分情况下, 更新订单数据只会更新 extend 和 status
	  extend 如果extend为空字符串则不会更新extend
	  status 订单状态
	  remark 备注
	*/
	UpdateOrderStatus(ctx context.Context, orderID string, extend string, status order_model.OrderStatus, remark string) error
	// 设置支付状态
	SetPayStatus(ctx context.Context, orderID, thirdPayOid string, payStatus byte, remark string) error
}

type Model struct {
	ID          uint   `db:"id"`
	OrderID     string `db:"oid"`      // 订单id
	OrderType   byte   `db:"o_type"`   // 订单类型
	OrderStatus byte   `db:"o_status"` // 订单状态

	PayType         byte   `db:"pay_type"`      // 支付类型
	PayStatus       byte   `db:"pay_status"`    // 支付状态
	PayAmount       uint32 `db:"pay_amount"`    // 支付金额, 单位分
	ThirdPayOrderID string `db:"third_pay_oid"` // 第三方支付订单id

	Uid    string `db:"uid"`    // 唯一标识一个用户
	Extend string `db:"extend"` // 和o_type相关的数据
	Remark string `db:"remark"` // 备注

	Ctime      time.Time `db:"ctime"`
	Utime      time.Time `db:"utime"`
	UpdateNums uint      `db:"update_nums"` //更新次数, 可防止utime相同时的异常
}
