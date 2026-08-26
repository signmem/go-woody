# go-woody
## 用途  
* 类似 dnsmasq 功能, 可支持高并发请求    
* 只为了 staging 环境中为某些特殊域名执行 DNS 本地劫持功能  

## go-woddy 说明  
* 主要用于对 powerdns 数据库执行增删查改功能  
* 提供 restFUL api 实现上述功能   
* 编译 go1.18 以上  

# 组件    
* web api (go-woody) (80端口)  
* pdns-4.9.2-1pdns.el7.x86_64 (5300端口)
* bind-9 (53端口) 

# 组件说明 (DNS 劫持功能接口) 
##  web api (go-woody)   
* 利用标准 net/http 库实现 api 接口功能   
* 访问 mysql DB, 实现对 powerdns 增删改查功能  

## powerdns 
* 兼容劫持主机名功能 ( /api/hosts/) 接口
* 标准 dns 权威管理，递归服务器模式进行 DNS 记录管理  ( /api/v2/ ) 接口

## bind-9  
* 因为有需要，要对每个 dns client 请求都做详细日志记录
* pdns-recursor 对日志不友好，因此改用 named 作为前端 DNS 应答主要程序
* 日志格式参考 service/named.conf 配置文件


# restFUL api 说明

* 主机名劫持功能   /api/hosts 接口 

| 功能 | method | example | 
| :-- | :-- |  :-- |
| 分页查询 | GET | curl 'http://localhost/api/hosts/?page=1&per_page=1000' |
| 所有查询 | GET | curl 'http://localhost/api/hosts/  |
| 独立查询 | GET | curl 'http://localhost/api/hosts/<id> |
| 增加记录 | POST | curl -H 'Content-Type: application/json' -d '{"hosts": [{"hostname":"terry.vclound.com", "ip":"1.1.1.1"}]}' http://localhost/api/hosts/ |
| 删除记录 | DELETE | curl -X DELETE http://localhost/api/hosts/10333 |
| 修改记录 | PUT | curl -X PUT -H 'Content-Type: application/json' -d '{"hostname":"terry.vclound.com", "ip":"2.2.2.2"}' http://localhost/api/hosts/1333 |


# 标准 dns 管理接口
* /api/v2/ 接口  

| 组件 | 功能 | method | example |
| :-- | :-- | :-- | :-- |
| 域名 | 主增加 | POST | curl -H 'Content-Type: application/json' -d '{"master":"", "domains":["163.com"]}'  http://localhost/api/v2/domains/ |
| 域名 | 从增加 | POST | curl -X POST -H 'Content-Type: application/json' -d '{"master":"10.189.20.49:5300", "domains":["163.com"]}'  http://localhost/api/v2/domains/ |
| 域名 | 删除 | DELETE | curl -X DELETE -H 'Content-Type: application/json' http://localhost/api/v2/domains/163.com |
| 域名 | 查询 | GET | curl -X GET -H 'Content-Type: application/json' http://localhost/api/v2/domains/ |
| 主机 | 增加 | POST | curl -H 'Content-Type: application/json' -d '{"hosts":[{"hostname":"terry.163.com", "ip":"10.199.22.241"}]}' http://localhost/api/v2/hosts/ |
| 主机 | 增加 | POST | curl -H 'Content-Type: application/json' -d '{"hosts":[{"hostname":"tsdb-zxjzd.vclound.com", "ip":"10.199.215.228"}, {"hostname":"git-build6-9zsbp.163.com", "ip":"10.199.217.41"}, {"hostname":"terry8-zgaqx.163.com", "ip":"10.199.217.139"}  ]}' http://localhost/api/v2/hosts/ |
| 主机 | 查询 | GET | 全量查询 GET /api/v2/hosts/ | 
| 主机 | 查询 | GET | 按照域名ID GET  /api/v2/hosts/domain_id/{id} |
| 主机 | 查询 | GET | 按主机ID  GET /api/v2/hosts/host_id/{id} |
| 主机 | 查询 | GET | 按主机 GET /api/v2/hosts/host_name/{hsotname} |
| 主机 | 查询 | GET | 按IP GET /api/v2/hosts/host_ip/{ipaddr} |
| 主机 | 删除 | DELETE | curl -X DELETE -H 'Content-Type: application/json' http://localhost/api/v2/hosts/host_id/69 |
| 主机 | 更新 | PUT | curl -X PUT -d '{"hostname":"felcon-rpmbuild6-xmawe.163.com", "ip":"10.189.21.12"}' -H 'Content-Type: application/json' http://localhost/api/v2/hosts/67 |
| SOA | 查询 | GET | curl -X GET  -H 'Content-Type: application/json'  http://localhost/api/v2/soa/  |
| SOA | 查询 | GET | 指定 DOMAIN_ID  GET  http://localhost/api/v2/soa/domain_id/{domain_id}  |
| SOA | 查询 | GET | 指定 DOMAIN GET  http://localhost/api/v2/soa/domain_name/{domain_name}  |
| SOA | 更新 | POST | curl -X POST -d '{"domain_id": 22}'  -H 'Content-Type: application/json' http://localhost/api/v2/soa/ |
| SOA | 更新 | POST | curl -X POST -d '{"domain_name": "163.com" }'  -H 'Content-Type: application/json' http://localhost/api/v2/soa/ |
| NOTIFY | 更新 | POST | curl -X POST -d '{"sync": true }'  -H 'Content-Type: application/json' http://localhost/api/v2/slave/ | 
| NOTIFY | 查询 | GET | curl -X GET  http://localhost/api/v2/slave/ |




