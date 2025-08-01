package tcloud

import (
	"cvmspot/utils"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/sirupsen/logrus"
	cam "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cam/v20190116"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	tag "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tag/v20180813"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type Client struct {
	RegionClients map[string]*AClient
	Cfg           *utils.Config
	Log           *logrus.Logger
}

// AClient 组合多个客户端
type AClient struct {
	CvmClient    *cvm.Client
	DnspodClient *dnspod.Client
	TagClient    *tag.Client
	VpcClient    *vpc.Client
	CamClient    *cam.Client
	Region       string
	Log          *logrus.Logger
}

// SecurityGroupRule 定义安全组规则
type SecurityGroupRule struct {
	Protocol    string
	Port        string
	CidrIp      string
	Action      string
	Description string
}

type CreateIns struct {
	Region                  string            // 地域
	InstanceChargeType      string            // 实例计费类型 SPOTPAID 竞价实例、PREPAID：预付费，即包年包月 、POSTPAID_BY_HOUR：按小时后付费、CDHPAID：独享子机（基于专用宿主机创建，宿主机部分的资源不收费）、CDCPAID：专用集群付费
	Zone                    string            // 可用区
	InstanceType            string            // 实例类型 SA2.MEDIUM4
	DiskType                string            // 硬盘类型 LOCAL_BASIC：本地硬盘 LOCAL_SSD：本地SSD硬盘 CLOUD_BASIC：普通云硬盘 CLOUD_SSD：SSD云硬盘 CLOUD_PREMIUM：高性能云硬盘 CLOUD_BSSD：通用型SSD云硬盘 CLOUD_HSSD：增强型SSD云硬盘 CLOUD_TSSD：极速型SSD云硬盘
	DiskSize                int64             // 硬盘容量
	ImageId                 string            // 镜像ID
	VpcId                   string            // 私有网络ID 绑定可用区 香港一区  vpc-ojeoqbu8 香港二区 vpc-rxwpov7g 香港三区 vpc-rxwpov7g
	SubnetId                string            // 私有网络子网ID 绑定可用区 香港一区 subnet-cw62hg13 香港二区 subnet-nit67s0b 香港三区 subnet-ck3276n5
	InternetChargeType      string            // 网络计费模式 BANDWIDTH_PREPAID：预付费按带宽结算、TRAFFIC_POSTPAID_BY_HOUR：流量按小时后付费、BANDWIDTH_POSTPAID_BY_HOUR：带宽按小时后付费、BANDWIDTH_PACKAGE：带宽包用户
	InternetMaxBandwidthOut int64             // 公网出宽带上限 单位：Mbps
	PublicIpAssigned        bool              // 是否分配公网IP
	InternetServiceProvider string            // 线路类型 CMCC：中国移动、CTCC：中国电信、CUCC：中国联通 BGP 三网
	IPv4AddressType         string            // 公网IP 类型 WanIP：普通公网IP、HighQualityEIP：精品 IP、AntiDDoSEIP：高防 IP
	InstanceCount           int64             // 购买实例数量
	InstanceName            string            // 实例名称
	Password                string            // 密码，不设置则邮件发送默认密码。Linux实例密码必须8到30位，至少包括两项[a-z]，[A-Z]、[0-9] 和 [( ) ` ~ ! @ # $ % ^ & *  - + = | { } [ ] : ; ' , . ? / ]中的特殊符号。Windows实例密码必须12到30位，至少包括三项[a-z]，[A-Z]，[0-9] 和 [( ) ` ~ ! @ # $ % ^ & * - + = | { } [ ] : ; ' , . ? /]中的特殊符号。
	KeyIds                  []string          // 密钥id 列表
	SecurityGroupIds        []*string         // 安全组 sg-hsowxvht
	SecurityService         bool              // 开启云安全服务。若不指定该参数，则默认开启云安全服务。
	MonitorService          bool              // 开启云监控服务。若不指定该参数，则默认开启云监控服务。
	AutomationService       bool              // 开启云自动化助手服务（TencentCloud Automation Tools，TAT）。若不指定该参数，则公共镜像默认开启云自动化助手服务，其他镜像默认不开启云自动化助手服务。
	ClientToken             string            // 用于保证请求幂等性的字符串。该字符串由客户生成，需保证不同请求之间唯一，最大值不超过64个ASCII字符。若不指定该参数，则无法保证请求的幂等性。
	HostName                string            // 主机名
	Tags                    map[string]string // 标签
	MaxPrice                string            // 竞价出价
	SpotInstanceType        string            // 竞价出价类型  竞价请求类型，当前仅支持类型：one-time
	MarketType              string            // 市场选项类型，当前只支持取值：spot
	DryRun                  bool              // 是否只预检此次请求。 true：发送检查请求，不会创建实例。检查项包括是否填写了必需参数，请求格式，业务限制和云服务器库存。 如果检查不通过，则返回对应错误码；如果检查通过，则返回RequestId.false（默认）：发送正常请求，通过检查后直接创建实例
	CoreCount               int64             // 决定启用的CPU物理核心数。
	ThreadPerCore           int64             // 每核心线程数。该参数决定是否开启或关闭超线程。 1 表示关闭超线程 2 表示开启超线程
	LaunchTemplateId        string            // 实例启动模板ID，通过该参数可使用实例模板中的预设参数创建实例。
	LaunchTemplateVersion   int64             // 实例启动模板版本号，若给定，新实例启动模板将基于给定的版本号创建
	DisableApiTermination   bool              // 实例销毁保护标志，表示是否允许通过api接口删除实例。取值范围： true：表示开启实例保护，不允许通过api接口删除实例  false：表示关闭实例保护，允许通过api接口删除实例
}

// 网络信息
type InternetAccessible struct {
	InternetChargeType      string // 网络计费类型 TRAFFIC_POSTPAID_BY_HOUR 按量收费
	InternetMaxBandwidthOut int64  // 带宽大小单位 mbps
}

type Instance struct {
	InstenceId         string              // 实例id
	InstanceName       string              // 实例名称
	Region             string              // 地域
	PublicIP           string              // 公网IP
	InstanceChargeType string              // 实例计费类型 SPOTPAID 竞价实例
	CPU                int64               // cpu 核数
	Memory             int64               // 内存 单位G
	InstanceState      string              // 实例状态 RUNNING 正在运行
	InstanceType       string              // 实例类型 SA2.MEDIUM2
	OsName             string              // 系统镜像名称
	InternetAccessible *InternetAccessible // 网络
	Zone               string              // 所在可用区
	DomainName         string              // 域名
	Password           string              // 密码
}

type SubVpcP struct {
	TagKey     *string
	TagVal     *string
	SubnetName *string
	CidrBlock  *string
	Zone       *string
	VpcId      *string
}

type DnsRecordP struct {
	Domain     *string
	SubDomain  *string
	RecordType *string
	RecordLine *string
	Value      *string
	TTL        *uint64
}

type DnsRcordR struct {
	PublicIp *string
	RecordId *uint64
}

// NewClientWithLogger 为每个地域创建腾讯云客户端(带日志记录器)
func NewClientWithLogger(cfg utils.Config, log *logrus.Logger) (*Client, error) {
	credential := common.NewCredential(cfg.TConfig.SecretId, cfg.TConfig.SecretKey)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"

	client := &Client{
		RegionClients: make(map[string]*AClient),
		Cfg:           &cfg,
		Log:           log,
	}

	for _, mgr := range cfg.IBManager {
		for _, region := range mgr.Instance.Regions {
			if !cfg.IsCli {
				log.Debugf("正在创建 %s 地域客户端", region)
			}

			// 创建CVM客户端
			cvmClient, err := cvm.NewClient(credential, region, cpf)
			if err != nil {
				return nil, fmt.Errorf("创建CVM客户端失败: %v", err)
			}

			// 创建DNSPod客户端
			dnspodCpf := profile.NewClientProfile()
			dnspodCpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
			dnspodClient, err := dnspod.NewClient(credential, "", dnspodCpf)
			if err != nil {
				return nil, fmt.Errorf("创建DNSPod客户端失败: %v", err)
			}

			// 创建Tag客户端
			tagCpf := profile.NewClientProfile()
			tagCpf.HttpProfile.Endpoint = "tag.tencentcloudapi.com"
			tagClient, err := tag.NewClient(credential, region, tagCpf)
			if err != nil {
				return nil, fmt.Errorf("创建Tag客户端失败: %v", err)
			}

			// 创建Vpc客户端
			vpcCpf := profile.NewClientProfile()
			vpcCpf.HttpProfile.Endpoint = "vpc.tencentcloudapi.com"
			vpcClient, err := vpc.NewClient(credential, region, vpcCpf)
			if err != nil {
				return nil, fmt.Errorf("创建Vpc客户端失败: %v", err)
			}

			// 创建 Cam 客户端
			camCpf := profile.NewClientProfile()
			camCpf.HttpProfile.Endpoint = "cam.tencentcloudapi.com"
			camClient, err := cam.NewClient(credential, region, camCpf)
			if err != nil {
				return nil, fmt.Errorf("创建Cam客户端失败: %v", err)
			}

			client.RegionClients[region] = &AClient{
				CvmClient:    cvmClient,
				DnspodClient: dnspodClient,
				TagClient:    tagClient,
				VpcClient:    vpcClient,
				CamClient:    camClient,
				Region:       region,
				Log:          log,
			}

			if cfg.Uin == "" {
				cfg.Uin, _ = client.RegionClients[region].GetUserUin()
			}

		}
	}
	if !cfg.IsCli {
		log.Debug("腾讯云所有地域客户端初始化成功！")
	}

	return client, nil
}

// GetSpotPrice 获取指定地域和实例类型的竞价实例价格
// 根据传入的地域和镜像ID，获取竞价实例的最小价格和对应的可用区
func (c *Client) GetSpotPrice(regions []string, imageId string) (float64, string, error) {
	// 创建一个map，用于存储每个地域的价格
	InsPrice := make(map[string]float64)
	// 遍历传入的地域
	for _, region := range regions {

		// 获取对应地域的CVM客户端
		aCli := c.RegionClients[region]

		// 打印查询地域的日志
		c.Log.Debugf("查询地域 %s 所有可用区价格", region)
		// 获取地域的所有可用区信息
		zoneInfo, err := aCli.getDescribeZones()
		if err != nil {
			// 如果获取失败，返回错误信息
			return 0, "", fmt.Errorf("获取竞价实例价格失败 %v", err)
		}
		// 遍历每个可用区
		for _, zone := range zoneInfo {
			// 获取可用区的价格
			aCli.getInstancePrice(*zone.Zone, imageId, "SPOTPAID", InsPrice)
		}
	}

	// 计算最小价格地域
	minValue := math.MaxFloat64
	zone := ""
	// 遍历InsPrice，找到最小价格和对应的可用区
	for key, value := range InsPrice {
		if value < minValue {
			minValue = value
			zone = key
		}
	}

	// 如果没有找到最小价格，返回错误信息
	if zone == "" {
		return 0, "", fmt.Errorf("价格获取失败")
	}
	// 打印最小价格和对应的可用区
	c.Log.Infof("实例价格最低的可用区是 %s ,最低价为 %v ", zone, minValue)
	// 返回最小价格和对应的可用区
	return minValue, zone, nil
}

// 查询可用区
func (a *AClient) getDescribeZones() ([]*cvm.ZoneInfo, error) {
	request := cvm.NewDescribeZonesRequest()
	response, err := a.CvmClient.DescribeZones(request)
	if err != nil {
		return nil, fmt.Errorf("查询可用区失败，错误：%v", err)
	}
	return response.Response.ZoneSet, nil
}

// InstanceChargeType  实例计费类型。 PREPAID：预付费，即包年包月 POSTPAID_BY_HOUR：按小时后付费 SPOTPAID：竞价付费
func (a *AClient) getInstancePrice(zone, imageId, instanceChargeType string, price map[string]float64) error {
	request := cvm.NewInquiryPriceRunInstancesRequest()
	request.Placement = &cvm.Placement{
		Zone: common.StringPtr(zone),
	}
	request.ImageId = common.StringPtr(imageId)
	request.InstanceChargeType = common.StringPtr(instanceChargeType)
	response, err := a.CvmClient.InquiryPriceRunInstances(request)
	if err != nil || response.Response.Price.InstancePrice.UnitPriceDiscount == nil {
		// err.(*errors.TencentCloudSDKError)
		return fmt.Errorf("查询实例价格失败，错误 %v", err)
	}
	price[zone] = *response.Response.Price.InstancePrice.UnitPriceDiscount
	return nil
}

// GetOrCreateSecurityGroup 存在则删除重新创建安全组
func (a *AClient) GetOrCreateSecurityGroup(tagKey, tagVal string, sc *utils.SecurityGroupConfig) (string, error) {
	req := vpc.NewDescribeSecurityGroupsRequest()

	req.Filters = []*vpc.Filter{
		{
			Name:   common.StringPtr("tag:" + tagKey),
			Values: common.StringPtrs([]string{tagVal}),
		},
	}

	// 返回的resp是一个DescribeSecurityGroupsResponse的实例，与请求对象对应
	resp, err := a.VpcClient.DescribeSecurityGroups(req)

	if err != nil {
		return "", fmt.Errorf("查询安全组失败: %v", err)
	}

	if *resp.Response.TotalCount > 0 {
		//return *resp.Response.SecurityGroupSet[0].SecurityGroupId, nil
		// 存在则先删除
		for _, sec := range resp.Response.SecurityGroupSet {
			a.delSec(sec.SecurityGroupId)
		}
	}

	egress := make([]*vpc.SecurityGroupPolicy, 0)
	ingress := make([]*vpc.SecurityGroupPolicy, 0)

	for _, rule := range sc.Rules {
		if rule.Type == "I" {
			ingress = append(ingress, &vpc.SecurityGroupPolicy{
				Protocol:          common.StringPtr(rule.Protocol),
				Port:              common.StringPtr(rule.Port),
				CidrBlock:         common.StringPtr(rule.CidrIp),
				Action:            common.StringPtr(rule.Action),
				PolicyDescription: common.StringPtr(rule.Description),
			})
		}
		if rule.Type == "E" {
			egress = append(egress, &vpc.SecurityGroupPolicy{
				Protocol:          common.StringPtr(rule.Protocol),
				Port:              common.StringPtr(rule.Port),
				CidrBlock:         common.StringPtr(rule.CidrIp),
				Action:            common.StringPtr(rule.Action),
				PolicyDescription: common.StringPtr(rule.Description),
			})
		}
	}
	tags := make([]*vpc.Tag, 0)
	tags = append(tags, &vpc.Tag{
		Key:   common.StringPtr(tagKey),
		Value: common.StringPtr(tagVal),
	})
	creReq := vpc.NewCreateSecurityGroupWithPoliciesRequest()
	// 创建安全组
	creReq.GroupName = common.StringPtr(sc.SecurityName)
	creReq.GroupDescription = common.StringPtr(sc.GroupDescription)
	creReq.Tags = tags
	creReq.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
		Egress:  egress,
		Ingress: ingress,
	}

	// 返回的resp是一个CreateSecurityGroupWithPoliciesResponse的实例，与请求对象对应
	response, err := a.VpcClient.CreateSecurityGroupWithPolicies(creReq)
	if err != nil {
		return "", fmt.Errorf("创建安全组失败: %v", err)
	}

	return *response.Response.SecurityGroup.SecurityGroupId, nil
}

// 删除安全组
func (a *AClient) delSec(SecurityGroupId *string) error {
	request := vpc.NewDeleteSecurityGroupRequest()

	request.SecurityGroupId = SecurityGroupId
	_, err := a.VpcClient.DeleteSecurityGroup(request)

	if err != nil {
		return fmt.Errorf("删除安全组失败: %v", err)
	}
	return nil
}

func (a *AClient) FindOrCreateVpc(tagKey, tagVal, vpcName, cidrBlock *string) (string, error) {
	// 查询带标签的VPC
	req := vpc.NewDescribeVpcsRequest()
	req.Filters = []*vpc.Filter{
		{
			Name:   common.StringPtr("tag:" + *tagKey),
			Values: []*string{tagVal},
		},
	}

	resp, err := a.VpcClient.DescribeVpcs(req)
	if err != nil {
		a.Log.Debugf("查询VPC失败: %v", err)
	} else if *resp.Response.TotalCount > 0 {
		return *resp.Response.VpcSet[0].VpcId, nil
	}

	// 创建新VPC
	createReq := vpc.NewCreateVpcRequest()
	createReq.VpcName = vpcName
	createReq.CidrBlock = cidrBlock
	createReq.Tags = []*vpc.Tag{
		{
			Key:   tagKey,
			Value: tagVal,
		},
	}

	createResp, err := a.VpcClient.CreateVpc(createReq)
	if err != nil {
		return "", fmt.Errorf("创建VPC失败: %v", err)
	}

	return *createResp.Response.Vpc.VpcId, nil
}

func (a *AClient) FindOrCreateSubnet(subVpcP *SubVpcP) (string, error) {
	// 查询带标签的子网
	req := vpc.NewDescribeSubnetsRequest()
	req.Filters = []*vpc.Filter{
		{
			Name:   common.StringPtr("vpc-id"),
			Values: []*string{subVpcP.VpcId},
		},
		{
			Name:   common.StringPtr("zone"),
			Values: []*string{subVpcP.Zone},
		},
		{
			Name:   common.StringPtr("tag:" + *subVpcP.TagKey),
			Values: []*string{subVpcP.TagVal},
		},
	}

	resp, err := a.VpcClient.DescribeSubnets(req)
	if err != nil {
		a.Log.Debugf("查询子网失败: %v", err)
	} else if *resp.Response.TotalCount > 0 {
		return *resp.Response.SubnetSet[0].SubnetId, nil
	}

	// 创建新子网
	createReq := vpc.NewCreateSubnetRequest()
	createReq.VpcId = subVpcP.VpcId
	createReq.SubnetName = subVpcP.SubnetName
	createReq.CidrBlock = subVpcP.CidrBlock
	createReq.Zone = subVpcP.Zone
	createReq.Tags = []*vpc.Tag{
		{
			Key:   subVpcP.TagKey,
			Value: subVpcP.TagVal,
		},
	}

	createResp, err := a.VpcClient.CreateSubnet(createReq)
	if err != nil {
		return "", fmt.Errorf("创建子网失败: %v", err)
	}

	return *createResp.Response.Subnet.SubnetId, nil
}

// 获取安全组和子网
func (a *AClient) GetOrCreateVpcAndSg(ibm *utils.InstanceBindingManager, zone, tagKey string) (string, string, string, error) {
	vpcId := ibm.Instance.VpcConfig.VpcId
	subnetId := ibm.Instance.SubnetConfig.SubnetId
	securityGroupId := ibm.Instance.SecurityGroups.SecurityGroupId
	var err error

	if vpcId == "" {
		vpcId, err = a.FindOrCreateVpc(&tagKey, &ibm.Instance.VpcConfig.TagVal, &ibm.Instance.VpcConfig.VpcName, &ibm.Instance.VpcConfig.CidrBlock)
		if err != nil {
			a.Log.Fatalf("创建VPC失败 %v", err)
		}
	}

	if subnetId == "" && vpcId != "" {
		subnetId, err = a.FindOrCreateSubnet(&SubVpcP{
			VpcId:      &vpcId,
			TagKey:     &tagKey,
			TagVal:     &ibm.Instance.SubnetConfig.TagVal,
			SubnetName: &ibm.Instance.SubnetConfig.SubnetName,
			CidrBlock:  &ibm.Instance.SubnetConfig.CidrBlock,
			Zone:       &zone,
		})
		if err != nil {
			a.Log.Fatalf("创建子网失败 %v", err)
		}
	}

	if securityGroupId == "" {
		securityGroupId, err = a.GetOrCreateSecurityGroup(tagKey, ibm.Instance.SecurityGroups.TagVal, &ibm.Instance.SecurityGroups)
		if err != nil {
			a.Log.Fatalf("创建安全组失败 %v", err)
		}
	}

	return vpcId, subnetId, securityGroupId, nil
}

func (a *AClient) RunInstances(ins *CreateIns) ([]*string, error) {

	req := cvm.NewRunInstancesRequest()
	req.InstanceType = common.StringPtr(ins.InstanceType)
	req.ImageId = common.StringPtr(ins.ImageId)
	req.InstanceChargeType = common.StringPtr(ins.InstanceChargeType)
	req.InstanceCount = common.Int64Ptr(ins.InstanceCount)
	req.Placement = &cvm.Placement{
		Zone: common.StringPtr(ins.Zone),
	}
	req.VirtualPrivateCloud = &cvm.VirtualPrivateCloud{
		VpcId:    common.StringPtr(ins.VpcId),
		SubnetId: common.StringPtr(ins.SubnetId),
	}
	// 设置SystemDisk
	req.SystemDisk = &cvm.SystemDisk{
		DiskType: common.StringPtr(ins.DiskType),
		DiskSize: common.Int64Ptr(ins.DiskSize),
	}
	// 设置InternetAccessible
	req.InternetAccessible = &cvm.InternetAccessible{
		InternetChargeType:      common.StringPtr(ins.InternetChargeType),
		InternetMaxBandwidthOut: common.Int64Ptr(ins.InternetMaxBandwidthOut),
	}

	// 设置默认InstanceChargeType
	req.InstanceChargeType = common.StringPtr(ins.InstanceChargeType)

	tags := []*cvm.Tag{}
	for k, v := range ins.Tags {
		tags = append(tags, &cvm.Tag{
			Key:   common.StringPtr(k),
			Value: common.StringPtr(v),
		})
	}

	// 设置默认TagSpecification
	req.TagSpecification = []*cvm.TagSpecification{
		{
			ResourceType: common.StringPtr("instance"),
			Tags:         tags,
		},
	}
	req.SecurityGroupIds = ins.SecurityGroupIds
	req.LoginSettings = &cvm.LoginSettings{
		Password: common.StringPtr(ins.Password),
	}

	// 详细日志记录
	a.Log.WithFields(logrus.Fields{
		"地域":    ins.Region,
		"实例类型":  req.InstanceType,
		"镜像ID":  req.ImageId,
		"私网ID":  req.VirtualPrivateCloud.VpcId,
		"子网ID":  req.VirtualPrivateCloud.SubnetId,
		"安全组ID": req.SecurityGroupIds,
		"可用区":   req.Placement.Zone,
		"磁盘类型":  req.SystemDisk.DiskType,
		"磁盘容量":  req.SystemDisk.DiskSize,
		"公网带宽":  req.InternetAccessible.InternetMaxBandwidthOut,
		"计费类型":  req.InstanceChargeType,
		"实例数量":  req.InstanceCount,
		"标签":    req.TagSpecification,
	}).Debug("创建实例请求参数")

	// 调用API创建
	resp, err := a.CvmClient.RunInstances(req)
	if err != nil {
		return nil, fmt.Errorf("实例创建失败: %v", err)
	}

	return resp.Response.InstanceIdSet, nil
}

// AddDNSRecord 添加DNS记录
func (a *AClient) AddDNSRecord(dp *DnsRecordP) error {
	req := dnspod.NewCreateRecordRequest()
	req.Domain = dp.Domain
	req.SubDomain = dp.SubDomain
	req.RecordType = dp.RecordType
	req.RecordLine = dp.RecordLine
	req.Value = dp.Value
	req.TTL = dp.TTL

	_, err := a.DnspodClient.CreateRecord(req)
	if err != nil {
		return fmt.Errorf("添加DNS记录失败: %v", err)

	}
	return nil
}

// getInstanceCount 获取实例数
// 参数说明（可选）
// 返回值说明（可选）
func (a *AClient) GetInstanceCount(tagKey, tagVal string) (int64, error) {
	req := cvm.NewDescribeInstancesRequest()
	// 通过标签过滤
	req.Filters = []*cvm.Filter{
		{
			Name:   common.StringPtr("tag:" + tagKey),
			Values: common.StringPtrs([]string{tagVal}),
		},
	}

	resp, err := a.CvmClient.DescribeInstances(req)
	if err != nil {
		return 0, fmt.Errorf("failed to describe instances: %v", err)
	}

	return *resp.Response.TotalCount, nil
}

func (a *AClient) GetInsInfo(tagKey, tagVal string) ([]*cvm.Instance, error) {
	req := cvm.NewDescribeInstancesRequest()
	req.Filters = []*cvm.Filter{
		{
			Name:   common.StringPtr("tag:" + tagKey),
			Values: []*string{common.StringPtr(tagVal)},
		},
	}

	resp, err := a.CvmClient.DescribeInstances(req)
	if err != nil {
		return nil, fmt.Errorf("获取实例列表错误: %v", err)
	}
	return resp.Response.InstanceSet, nil
}

// TransferFiles 传输文件到实例
func (c *Client) TransferFiles(instanceId, localPath, remotePath string) error {
	cmd := exec.Command("scp", "-r", localPath, fmt.Sprintf("root@%s:%s", instanceId, remotePath))
	return cmd.Run()
}

// ExecuteCommands 在实例上执行命令
func (c *Client) ExecuteCommands(instanceId string, commands []string, logPath string) (string, error) {
	cmdStr := ""
	for _, cmd := range commands {
		cmdStr += cmd + "; "
	}
	cmd := exec.Command("ssh", fmt.Sprintf("root@%s", instanceId), cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	if logPath != "" {
		cmd = exec.Command("ssh", fmt.Sprintf("root@%s", instanceId),
			fmt.Sprintf("echo '%s' >> %s", string(output), logPath))
		err = cmd.Run()
		if err != nil {
			return string(output), err
		}
	}
	return string(output), nil
}

// RandomDelete 随机删除指定数量的实例
func (a *AClient) RandomDelete(count int64) error {
	if count <= 0 {
		return nil
	}

	// 获取所有实例
	req := cvm.NewDescribeInstancesRequest()
	resp, err := a.CvmClient.DescribeInstances(req)
	if err != nil {
		return fmt.Errorf("获取实例列表失败: %v", err)
	}

	instances := resp.Response.InstanceSet
	if len(instances) == 0 {
		return nil
	}

	// 随机选择实例
	rand.Shuffle(len(instances), func(i, j int) {
		instances[i], instances[j] = instances[j], instances[i]
	})

	// 限制删除数量不超过实例总数
	if count > int64(len(instances)) {
		count = int64(len(instances))
	}

	// 删除选中的实例
	for i := int64(0); i < count; i++ {
		instanceId := instances[i].InstanceId
		delReq := cvm.NewTerminateInstancesRequest()
		delReq.InstanceIds = []*string{instanceId}
		_, err := a.CvmClient.TerminateInstances(delReq)
		if err != nil {
			a.Log.Errorf("删除实例 %s 失败: %v", *instanceId, err)
			continue
		}
		a.Log.Infof("成功删除实例 %s", *instanceId)
	}

	return nil
}

func (a *AClient) RemoveDNSRecord(domain *string, recordId *uint64) error {
	request := dnspod.NewDeleteRecordRequest()

	request.Domain = domain
	request.RecordId = recordId
	// 返回的resp是一个DeleteRecordResponse的实例，与请求对象对应
	_, err := a.DnspodClient.DeleteRecord(request)

	if err != nil {
		return fmt.Errorf("记录删除失败 %v", err)
	}
	return nil
}

func (c *AClient) GetDnsRecordList(Domain, Subdomain *string) ([]*DnsRcordR, error) {
	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := dnspod.NewDescribeRecordListRequest()

	request.Domain = Domain
	request.Subdomain = Subdomain
	// 返回的resp是一个DescribeRecordListResponse的实例，与请求对象对应
	response, err := c.DnspodClient.DescribeRecordList(request)
	if err != nil {
		return nil, fmt.Errorf("获取域名 %s.%s 解析信息失败 %v", *Subdomain, *Domain, err)
	}
	drList := make([]*DnsRcordR, 0)

	for _, record := range response.Response.RecordList {
		if *record.Type == "A" {
			drList = append(drList, &DnsRcordR{
				RecordId: record.RecordId,
				PublicIp: record.Value,
			})
		}
	}
	return drList, nil

}

func (c *Client) GetInsMap(insMap *map[string]*Instance) {
	*insMap = make(map[string]*Instance)

	for _, mgr := range c.Cfg.IBManager {
		for _, region := range mgr.Instance.Regions {
			instanceSet, _ := c.RegionClients[region].GetInsInfo(c.Cfg.TConfig.TagKey, mgr.Name)
			for _, instance := range instanceSet {
				// 提取所有 IP 地址（处理可能的 nil 指针）
				ips := make([]string, 0, len(instance.PublicIpAddresses))
				for _, ipPtr := range instance.PublicIpAddresses {
					ips = append(ips, *ipPtr) // 解引用
				}

				ipStr := strings.Join(ips, ",")
				resIpStr := "N/A"
				if ipStr != "" {
					resIpStr = ipStr
				}

				(*insMap)[*instance.InstanceId] = &Instance{
					InstenceId:         *instance.InstanceId,
					InstanceName:       *instance.InstanceName,
					Region:             region,
					PublicIP:           resIpStr,
					InstanceChargeType: *instance.InstanceChargeType,
					CPU:                *instance.CPU,
					Memory:             *instance.Memory,
					InstanceState:      *instance.InstanceState,
					InstanceType:       *instance.InstanceType,
					OsName:             *instance.OsName,
					InternetAccessible: &InternetAccessible{
						InternetChargeType:      *instance.InternetAccessible.InternetChargeType,
						InternetMaxBandwidthOut: *instance.InternetAccessible.InternetMaxBandwidthOut,
					},
					Zone:       *instance.Placement.Zone,
					DomainName: "N/A",
				}
			}

			// 查询实例标签
			rows, err := c.RegionClients[region].GetTag(mgr.DomainBinding.TagKey, "")
			if err != nil {
				c.Log.Errorln(err)
			} else {
				if len(*insMap) > 0 {
					for _, res := range rows {
						if *res.ServiceType == "cvm" && *res.ResourcePrefix == "instance" && *res.ResourceRegion == region {
							for _, tag := range res.Tags {
								if *tag.TagKey == mgr.DomainBinding.TagKey {
									(*insMap)[*res.ResourceId].DomainName = *tag.TagValue
								}
							}
						}

					}
				}

			}

		}

	}

}

func getTableType(t int) *tablewriter.Table {
	var table *tablewriter.Table
	switch t {
	case 1:
		table = tablewriter.NewTable(os.Stdout,
			tablewriter.WithRenderer(renderer.NewMarkdown()),
		)
	case 2:
		symbols := tw.NewSymbolCustom("Nature").
			WithRow("~").
			WithColumn("|").
			WithTopLeft("🌱").
			WithTopMid("🌿").
			WithTopRight("🌱").
			WithMidLeft("🍃").
			WithCenter("❀").
			WithMidRight("🍃").
			WithBottomLeft("🌻").
			WithBottomMid("🌾").
			WithBottomRight("🌻")

		table = tablewriter.NewTable(os.Stdout, tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{Symbols: symbols})))

	case 3:
		table = tablewriter.NewTable(os.Stdout)
	}
	return table
}

func (c *Client) ListInstances() {
	insMap := make(map[string]*Instance)
	c.GetInsMap(&insMap)

	if len(insMap) == 0 {
		fmt.Println("---未查到腾讯云实例---")
		return
	}

	// 打印表格
	table := getTableType(1)
	data := [][]string{}
	for _, instance := range insMap {
		data = append(data, []string{
			instance.InstenceId,
			instance.Zone,
			instance.PublicIP,
			instance.DomainName,
			instance.InstanceType,
		})
	}

	table.Header([]string{"ID", "可用区", "公网IP", "域名", "实例类型"})
	table.Bulk(data)
	table.Render()
}

// insMap key 为 实例ID value 为 region
func (c *Client) DeleteInstances(instanceIDs []string) {

	if len(instanceIDs) == 0 {
		fmt.Printf("必须指定至少一个实例ID: %v \n", instanceIDs)
		return
	}

	insMap := make(map[string]*Instance)
	c.GetInsMap(&insMap)

	// 创建一个map来保存结果
	insIdToReg := make(map[string][]*string)

	// 遍历instanceIDs，根据region分组
	for _, id := range instanceIDs {
		if ins, exists := insMap[id]; exists {
			// 获取instanceID的指针
			ptr := &id
			// 将指针添加到对应region的切片中
			insIdToReg[ins.Region] = append(insIdToReg[ins.Region], ptr)
		}
	}

	for region, instanceIds := range insIdToReg {
		req := cvm.NewTerminateInstancesRequest()
		req.InstanceIds = instanceIds
		_, err := c.RegionClients[region].CvmClient.TerminateInstances(req)
		if err != nil {
			fmt.Printf("删除实例错误: %v \n", err)
		}
		fmt.Printf("成功删除实例: %v \n", instanceIDs)
	}
}

// 获取用户UIN
func (a *AClient) GetUserUin() (string, error) {
	request := cam.NewGetUserAppIdRequest()
	response, err := a.CamClient.GetUserAppId(request)
	if err != nil {
		return "", fmt.Errorf("获取用户Uin失败: %v", err)
	}
	return *response.Response.Uin, nil
}

// 添加标签
func (c *AClient) AddTag(tagKey, tagVal, region, Uin, insId string) error {
	request := tag.NewAddResourceTagRequest()

	request.TagKey = &tagKey
	request.TagValue = &tagVal
	request.Resource = common.StringPtr("qcs::cvm:" + region + ":uin/" + Uin + ":instance/" + insId)
	// 返回的resp是一个AddResourceTagResponse的实例，与请求对象对应
	_, err := c.TagClient.AddResourceTag(request)
	if err != nil {
		return fmt.Errorf("添加标签失败: %v", err)
	}
	return nil
}

// 查询标签
func (c *AClient) GetTag(tagKey, tagVal string) ([]*tag.ResourceTag, error) {
	request := tag.NewDescribeResourcesByTagsRequest()

	if tagVal == "" {
		request.TagFilters = []*tag.TagFilter{
			{
				TagKey: common.StringPtr(tagKey),
			},
		}
	} else {
		request.TagFilters = []*tag.TagFilter{
			{
				TagKey:   common.StringPtr(tagKey),
				TagValue: common.StringPtrs([]string{tagVal}),
			},
		}
	}

	// 返回的resp是一个DescribeResourcesByTagsResponse的实例，与请求对象对应
	response, err := c.TagClient.DescribeResourcesByTags(request)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %v", err)
	}
	return response.Response.Rows, nil
}
