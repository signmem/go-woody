# go-woody
## 用途  
* 类似 dnsmasq 功能, 可支持高并发请求    
* 只为了 staging 环境中为某些特殊域名执行 DNS 本地劫持功能  

## go-woddy 说明  
* 主要用于对 powerdns 数据库执行增删查改功能  
* 提供 restFUL api 实现上述功能   
* 编译 go1.18 以上  

# 组件    
* web api (go-woody)   
* pdns-3.4.11-1.el7 (53 端口)  
* pdns-recursor-3.7.4-1.el7 (5300 端口) 

# 组件说明  
##  web api (go-woody)   
* 利用标准 net/http 库实现 api 接口功能   
* 访问 mysql DB, 实现对 powerdns 增删改查功能  

## powerdns 
* 使用 3.4 版本, 可以避免增加 pdns-recursor 服务器中对 zone (定义本地域解析) 文件管理  
* 不使用标准权威，递归服务器模式进行 DNS 记录管理   
* 例如，希望对  sports.news.163.com 执行本地解析, 如果执行递归解析，则所有 163.com 域名都需要本地解析  

## powerdns-recurosr  
* 本地无法解析域名都由 recursor 完成 forwards 解析   
* 使用 3.4 版本，避免管理 zone 文件并需要 reload 服务  



# restFUL api 说明

* 统一使用 api api/hosts 接口 

| 功能 | method | example | 
| :-- | :-- |  :-- |
| 分页查询 | GET | curl 'http://localhost/api/hosts/?page=1&per_page=1000' |
| 所有查询 | GET | curl 'http://localhost/api/hosts/ \<br>默认一次1000页 |
| 独立查询 | GET | curl 'http://localhost/api/hosts/<id> |
| 增加记录 | POST | curl -H 'Content-Type: application/json' \<br>-d '{"hosts": [{"hostname":"terry.vclound.com", "ip":"1.1.1.1"}]}' \<br>http://localhost/api/hosts/ |
| 删除记录 | DELETE | curl -X DELETE http://localhost/api/hosts/10333 |
| 修改记录 | PUT | curl -X PUT \<br>-H 'Content-Type: application/json' \<br>-d '{"hostname":"terry.vclound.com", "ip":"2.2.2.2"}' \<br>http://localhost/api/hosts/1333 |


