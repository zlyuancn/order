create table order_0
(
    id            int unsigned auto_increment
        primary key,
    oid           varchar(128)     default ''                  not null comment '订单id',
    o_type        tinyint unsigned default 0                   not null comment '订单类型',
    o_status      tinyint unsigned default 1                   not null comment '订单状态',

    pay_type      tinyint unsigned default 0                   not null comment '支付类型',
    pay_status    tinyint unsigned default 0                   not null comment '支付状态',
    pay_amount    int unsigned     default 0                   not null comment '付费金额, 单位分',
    third_pay_oid varchar(128)     default ''                  not null comment '第三方支付订单id',

    uid           varchar(128)     default ''                  not null comment '用户唯一标识',
    extend        varchar(8192)    default '{}'                not null comment '和o_type相关的数据',
    remark        varchar(1024)    default ''                  not null comment '备注',

    ctime         datetime         default current_timestamp() not null,
    utime         datetime         default current_timestamp() not null,
    update_nums   int unsigned     default 0                   not null comment '更新次数, 可防止utime相同时的异常',
    constraint oid_index
        unique (oid)
)
    comment '订单流水号';

create index uid_index on order_0 (uid);
create index third_pay_oid_index on order_0 (third_pay_oid);


create table order_1
(
    id            int unsigned auto_increment
        primary key,
    oid           varchar(128)     default ''                  not null comment '订单id',
    o_type        tinyint unsigned default 0                   not null comment '订单类型',
    o_status      tinyint unsigned default 1                   not null comment '订单状态',

    pay_type      tinyint unsigned default 0                   not null comment '支付类型',
    pay_status    tinyint unsigned default 0                   not null comment '支付状态',
    pay_amount    int unsigned     default 0                   not null comment '付费金额, 单位分',
    third_pay_oid varchar(128)     default ''                  not null comment '第三方支付订单id',

    uid           varchar(128)     default ''                  not null comment '用户唯一标识',
    extend        varchar(8192)    default '{}'                not null comment '和o_type相关的数据',
    remark        varchar(1024)    default ''                  not null comment '备注',

    ctime         datetime         default current_timestamp() not null,
    utime         datetime         default current_timestamp() not null,
    update_nums   int unsigned     default 0                   not null comment '更新次数, 可防止utime相同时的异常',
    constraint oid_index
        unique (oid)
)
    comment '订单流水号';

create index uid_index on order_1 (uid);
create index third_pay_oid_index on order_1 (third_pay_oid);


