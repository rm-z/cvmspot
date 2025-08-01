# README

# 1.腾讯云竞价实例自动创建

 支持自动创建竞价实例的云服务器，当竞价实例被释放时可自动创建。

# 2.快速开始

```shell
# 2.1 下载项目
git clone https://github.com/rm-z/cvmspot.git

# 2.2 官网下载配置go1.24.5环境
https://golang.google.cn/doc/install

# 2.3 安装依赖包
go mod tidy

# 2.4 运行项目
cd cvmspot
go run main.go

# 2.5 打包
# 打包为 Windows amd64 版本
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o cvmspot.exe main.go

# 2.6 启动项目
# 将配置文件 config-template.yaml 重命名为 config.yaml ,将其与 cvmspot.exe 程序放同目录下
# 保证环境变量存在 TENCENTCLOUD_SECRET_ID 和 TENCENTCLOUD_SECRET_KEY 或在配置文件添加 secret_id 和 secret_key 配置
# 2.6.1 Windows 不加任何参数将以服务的方式启动项目，将会根据配置文件自动创建实例，并保持配置的实例数量
cvmspot.exe

# 2.6.2 Windows 以客户端方式执行
# 查看以此程序创建的腾讯云实例
cvmspot.exe cvm -l

# 2.6.3 删除此程序创建的腾讯云实例，支持传入一个或多个实例ID,实例ID可由 cvmspot.exe cvm -l 查询
cvmspot.exe cvm -d 实例ID_1 实例ID_2
```

## 3.配置示例

```yaml
# 腾讯云配置
# 密钥可前往官网控制台 https://console.cloud.tencent.com/cam/capi 进行获取
# 支持在环境变量配置，优先从环境变量获取 TENCENTCLOUD_SECRET_ID 和 TENCENTCLOUD_SECRET_KEY
# tag_key 标签Key，判断实例、安全组、私有网络等是否由此程序创建
tencentcloud:
    secret_id: 
    secret_key: 
    tag_key: fromAutoCvmSpot

# 日志配置
log:
    log_path: ./cvmspot.log
    level: debug

# 实例管理器组，每个成员配置相互独立
instance_managers:
    # 实例管理器
    - name: spot-instance-group1
      instance:
        instance_name: cvmspot-01
        # 实例镜像ID，官方镜像ID 查询 https://cloud.tencent.com/document/product/213/93093
        image_id: img-l8og963d
        # 实例类型（标准SA2一般为最便宜类型），参考官方 https://cloud.tencent.com/document/product/213/11518
        instance_type: SA2.MEDIUM4
        # 实例计费模式，SPOTPAID 竞价实例、PREPAID：预付费，即包年包月 、POSTPAID_BY_HOUR：按小时后付费
        internet_charge_type: SPOTPAID
        # 宽带
        internet:
            # 带宽大小单位 mbps
            bandwidth_out: 100
            # 按量计费
            charge_type: TRAFFIC_POSTPAID_BY_HOUR
        # 地域，此实例管理器创建实例所在地域
        # 地域列表 https://cloud.tencent.com/document/api/213/15692
        regions:
            - ap-hongkong
        # 安全组
        # 指定对应地域的安全组ID，security_groupId 则会使用对应安全组
        # 不存在会根据 出入站规则 rules 创建安全组，每次重新创建实例会根据出入站规则重新创建安全组
        security_groups:
            security_groupId: 
            security_name: byCvmSpot
            group_description: byCvmSpot
            tag_val: sc
            # 出入站规则
            rules: 
              # type I 入站 E 出站 IE 入站和出站
              - type: I
                # 端口类型支持 tcp/udp/all 或 ICMP, ICMPv6, GRE 参考腾讯云安全组规则
                protocol: tcp
                # 支持 单个端口 80 ，多个端口 80，443 端口段 4000-5000 ，如果 protocol 是all ，port 也要为all
                port: 22
                # 来源IP，允许那些ip访问此机器，支持单个IP，CIDR，IP段，0.0.0.0/0 表示所有IP
                cidr_ip: 0.0.0.0/0
                # 规则动作 ACCEPT 允许，DROP 拒绝
                action: ACCEPT
                # 描述
                desc: ssh
              - type: I
                protocol: tcp
                port: 7000
                cidr_ip: 0.0.0.0/0
                action: ACCEPT
                desc: frp
              - type: I
                protocol: tcp
                port: 4000-5000
                cidr_ip: 0.0.0.0/0
                action: ACCEPT
                desc: frps port
              - type: I
                protocol: udp
                port: 8211
                cidr_ip: 0.0.0.0/0
                action: ACCEPT
                desc: hspl
              - type: E
                protocol: all
                port: all
                cidr_ip: 0.0.0.0/0
                action: ACCEPT
                desc: ssh
        # 实例磁盘配置 CLOUD_PREMIUM 高性能云硬盘 一般为最便宜硬盘
        # 参考硬盘类型 https://cloud.tencent.com/document/product/362/2353
        system_disk:
            # 容量 单位G
            size: 20
            # 硬盘类型
            type: CLOUD_PREMIUM
        # 私有网络，指定私有网络id vpc_id 则使用对应网络
        # 不指定则根据配置自动创建
        vpc:
          tag_val: byCvmSpot
          vpc_id: 
          Vpc_name: vpc-cvmspot
          # 私网IP段
          cidr_block: 10.0.0.0/12
        subnet:
          tag_val: byCvmSpot
          subnet_id: 
          subnet_name: vpc-cvmspot
          # 子网IP段
          cidr_block: 10.0.n.0/24
        user:
          # 请手动配置所选镜像的默认用户名
          # 不同类型镜像为不同默认账号，如 Centos 为 root ，ubuntu 为 ubuntu
          # 不进行自动化上传和执行任务，也不想执行实例密码，请忽略此配置
          username: root
          password: xfdetk@s.d12gjs
      # 自动化相关配置
      auto_maintenance:
        # 是否要自动创建实例
        enabled: true
        # 实例检测间隔，单位秒
        check_interval: 60
        # 要创建实例数量
        desired_count: 1
        # 能接受的最高实例单价
        lowest_price: 0.05
        # 实例数量高于 desired_count 是否自动删除
        auto_remove: false

      # 绑定域名配置 （只支持 DnsPod， 且与实例在同一账号下）
      domain_binding:
        # 是否自动给实例公网IP 绑定域名
        enabled: true
        tag_key: domain_name
        # 子域名支持的最大解析Ip数量
        prase_num: 2
        # 主域名
        domain: test.com
        record_line: 默认
        record_type: A
        # 二级域名
        subdomain: frp
        ttl: 600
      # 实例创建完成后的自动化操作
      feature:
        # 文件上传
        file_transfer:
            enabled: true
            # 本地路径
            local_path: ./scripts
            # 远程路径（不存在则会创建）
            remote_path: /root
        command_exec:
            enabled: true
            # 上传完后执行什么命令（未配置上传会直接执行）
            command: sh ~/install_frp.sh     


```
