
# 前置准备

## mysql

1. 首先准备一个库名为 `order` 的mysql库. 这个库名可以通过配置`DBName`修改
2. 创建订单的分表, 默认为2个分表, 分表索引从0开始, 可以通过配置`TableShardNums`修改.
   1. 构建分表的工具为 [stf](https://github.com/zlyuancn/stt/tree/master/stf)
   2. 订单系统的分表文件在[这里](https://github.com/zlyuancn/order/tree/master/db_table/order_.sql)
   3. 在[这里](https://github.com/zlyuancn/order/tree/master/db_table/order_.out.sql)可以看到已经生成好了2个分表的sql文件, 可以直接导入.

# 订单从创建到付款到发货基础流程, 使用者只开发关注业务层代码(下图粉色部分)

```mermaid
sequenceDiagram
participant u as 用户
participant a as 业务层
participant b as order平台
participant f as 第三方付费平台

a ->> b: 注册业务 (RegistryOrderBusiness)

opt 用户预付费下单
rect rgb(230, 250, 255)
u ->> a: 下单
    rect rgb(250, 180, 220)
    a ->> b: 生成订单号 (GenOID)
    a ->> b: 下单 (CreateOrder) 并启用后置补偿 (enableCompensation=true)
    a ->> b: 推进订单 (Forward)
    end
a -->> u: rps ok
end
end

opt 等待用户付费下单
rect rgb(230, 250, 255)
u ->> a: 下单
    rect rgb(250, 180, 220)
    a ->> b: 生成订单号 (GenOID)
    a ->> b: 下单 (CreateOrder) 不启用后置补偿 (enableCompensation=false)
    end
a -->>+ u: rps ok

u ->>- f: 用户付费

rect rgb(200, 240, 255)
opt 付费回调处理
f -->> a: 付费完成回调
    rect rgb(250, 180, 220)
    a ->> b: 更新付费状态 (UpdatePayStatus)
    a ->> b: 发送后置补偿信号 (SendCompensationSignal)
    end
a -->> f: ok
end

par 启协程推进订单
    rect rgb(250, 180, 220)
    a ->> b: 推进订单 (Forward), 此处失败不用告知第三方付费平台失败
    end
a -->> a: 订单完成之后的其它处理
end
end

end
end
```


## 完整的流程如下, 黄色部分表示order平台工作

```mermaid
sequenceDiagram
participant u as 用户
participant a as 业务层
participant b as order平台
participant c as mysql
participant d as redis
participant e as mq
participant f as 第三方付费平台

a ->> b: 注册业务 (RegistryOrderBusiness)

opt 用户预付费下单
rect rgb(230, 250, 255)
u ->> a: 下单
a ->> b: 生成订单号 (GenOID)
rect rgb(250, 250, 220)
b ->> d: incr生成订单号
end
a ->> b: 下单 (CreateOrder) 并启用后置补偿 (enableCompensation=true)
rect rgb(250, 250, 220)
b ->> e: 写入补偿mq
b ->> c: 写入订单数据
end
a ->> b: 推进订单 (Forward)

rect rgb(250, 250, 220)
b ->> d: 加订单锁
b -->> c: 获取订单数据
b ->> b: 扣款
b ->> b: 发货
b ->> d: 解除订单锁
end

a -->> u: rsp ok
end
end

opt 等待用户付费下单
rect rgb(230, 250, 255)
u ->> a: 下单
a ->> b: 生成订单号 (GenOID)
rect rgb(250, 250, 220)
b ->> d: incr生成订单号
end
a ->> b: 下单 (CreateOrder) 不启用后置补偿 (enableCompensation=false)
b ->> c: 写入订单数据
a -->>+ u: rps ok

u ->>- f: 用户付费

rect rgb(200, 240, 255)
opt 付费回调处理
f -->> a: 付费完成回调
a ->> b: 更新付费状态 (UpdatePayStatus)

rect rgb(250, 250, 220)
b ->> c: 更新付费状态
end

a ->> b: 发送后置补偿信号 (SendCompensationSignal)

rect rgb(250, 250, 220)
b ->> e: 写入补偿mq
end

a -->> f: rsp ok
end

par 启协程推进订单
a ->> b: 推进订单 (Forward), 此处失败不用告知第三方付费平台失败
    rect rgb(250, 250, 220)
    b ->> d: 加订单锁
    b -->> c: 获取订单数据
    b ->> b: 扣款
    b ->> b: 发货
    b ->> d: 解除订单锁
    end
a -->> a: 订单完成之后的其它处理
end

end

end
end
```
