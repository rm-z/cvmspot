package service

import (
	"context"
	"cvmspot/tcloud"
	"cvmspot/utils"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type InstanceManager struct {
	Cfg      *utils.Config
	Ibm      *utils.InstanceBindingManager
	Log      *logrus.Logger
	Client   *tcloud.AClient
	InsCfg   *tcloud.CreateIns
	Region   string
	Zone     string
	Interval time.Duration
}

type InstanceManagerGroup struct {
	managers []*InstanceManager
	log      *logrus.Logger
	client   *tcloud.Client
}

// 标签描述列表。通过指定该参数可以同时绑定标签到相应的云服务器、云硬盘实例。
// type TagSpecification struct {
// 	ResourceType  string // 标签绑定的资源类型，云服务器为“instance”，专用宿主机为“host”，镜像为“image”，密钥为“keypair”，置放群组为“ps”，高性能计算集群为“hpc”
// 	Tags
// }

func NewInstanceManagerGroup(c *tcloud.Client, cfg *utils.Config) *InstanceManagerGroup {

	c.Log.Debugf("正在初始化实例管理器组...")

	// 初始化实例管理器组
	group := &InstanceManagerGroup{
		log:    c.Log, // 使用任意区域的CvmClient中的Log
		client: c,
	}

	for _, ibm := range cfg.IBManager {
		if ibm.AutoMaintenance.Enabled {
			c.Log.Infof("正在查询最低价实例所在可用区")
			// 获取最低价的实例可用区
			_, zone, err := c.GetSpotPrice(ibm.Instance.Regions, ibm.Instance.ImageId)
			if err != nil {
				c.Log.Fatalf("获取低价可用区失败，退出创建 %v", err)
				continue
			}

			// 查询私网ID
			aCli := c.RegionClients[zone[:len(zone)-2]]
			ibm.Instance.SubnetConfig.CidrBlock = strings.Replace(ibm.Instance.SubnetConfig.CidrBlock, "n", zone[len(zone)-1:], -1)
			vpcId, subnetId, sid, err := aCli.GetOrCreateVpcAndSg(&ibm, zone, cfg.TConfig.TagKey)
			if err != nil {
				c.Log.Fatalf("%v", err)
			} else {
				c.Log.WithFields(logrus.Fields{
					"私网ID":  vpcId,
					"子网ID":  subnetId,
					"安全组ID": sid,
				}).Info("获取私有网络和安全组成功")
			}

			group.managers = append(group.managers, &InstanceManager{
				Cfg:    cfg,
				Ibm:    &ibm,
				Log:    c.Log,
				Client: aCli,
				InsCfg: &tcloud.CreateIns{
					Region:                  zone[:len(zone)-2],
					InstanceChargeType:      ibm.Instance.InternetChargeType,
					Zone:                    zone,
					ImageId:                 ibm.Instance.ImageId,
					InstanceType:            ibm.Instance.InstanceType,
					DiskType:                ibm.Instance.SystemDisk.Type,
					DiskSize:                ibm.Instance.SystemDisk.Size,
					VpcId:                   vpcId,
					SubnetId:                subnetId,
					InternetChargeType:      ibm.Instance.Internet.ChargeType,
					InternetMaxBandwidthOut: ibm.Instance.Internet.BandwidthOut,
					InstanceCount:           ibm.AutoMaintenance.DesiredCount,
					InstanceName:            ibm.Instance.InstanceName,
					SecurityGroupIds:        []*string{&sid},
					Tags:                    map[string]string{cfg.TConfig.TagKey: ibm.Name, ibm.DomainBinding.TagKey: ibm.DomainBinding.SubDomain + "." + ibm.DomainBinding.Domain},
					MaxPrice:                ibm.AutoMaintenance.LowestPrice,
					Password:                ibm.Instance.UserConfig.Password,
				},
				Region:   zone[:len(zone)-2],
				Zone:     zone,
				Interval: time.Duration(ibm.AutoMaintenance.CheckInterval) * time.Second,
			})
		}
	}

	return group
}

func (g *InstanceManagerGroup) Run(ctx context.Context) {
	for _, mgr := range g.managers {
		go mgr.Run(ctx)
	}
}

// 运行 实例管理器
// Run 启动实例管理器的主循环
func (m *InstanceManager) Run(ctx context.Context) {
	m.Log.WithFields(logrus.Fields{
		"实例管理器": m.Ibm.Name,
		"区域":    m.Region,
		"可用区":   m.Zone,
	}).Info("实例管理器启动")

	// 立即执行首次检查
	m.Log.Debug("执行首次实例检查")
	m.checkIns()

	// 创建定时检查的ticker
	ticker := time.NewTicker(m.Interval)
	defer func() {
		ticker.Stop()
		m.Log.Info("实例管理器已停止")
	}()

	for {
		select {
		case <-ctx.Done():
			m.Log.Info("收到停止信号，实例管理器正在退出...")
			return
		case <-ticker.C:
			m.Log.Debug("正在检查实例状态...")
			start := time.Now()
			m.checkIns()
			m.Log.WithField("耗时", time.Since(start).Seconds()).Debug("实例检查完成")
		}
	}
}

// syncDNSRecords 同步DNS记录
func (m *InstanceManager) syncDNSRecords() (map[string]string, error) {
	var instanceSet []*cvm.Instance
	var err error
	loopNum := 1
	var currentIPs map[string]string // 声明变量但不初始化

	// 获取实例列表
	for {
		instanceSet, err = m.Client.GetInsInfo(m.Cfg.TConfig.TagKey, m.Ibm.Name)
		if err != nil {
			m.Log.Errorf("获取实例信息失败: %v", err)
			return nil, err
		}

		if len(instanceSet) >= int(m.Ibm.AutoMaintenance.DesiredCount) {
			// 收集当前实例的所有公网IP
			currentIPs = make(map[string]string, 0)
			for _, instance := range instanceSet {
				for _, ip := range instance.PublicIpAddresses {
					if ip != nil {
						currentIPs[*ip] = *instance.InstanceId
					}
				}
			}

			if len(currentIPs) >= int(m.Ibm.AutoMaintenance.DesiredCount) {
				m.Log.Debugf("已存在 %d 个腾讯云实例，开始检测并添加DNS记录", len(instanceSet))
				break
			}

		}

		if loopNum >= 10 {
			return nil, fmt.Errorf("等待腾讯云实例创建超时....%s", "")
		}

		m.Log.Debugf("正在进行第 %d/%d 次等待腾讯云创建实例并分配公网完成...", loopNum, 10)
		loopNum++

		time.Sleep(time.Second * 5)

	}

	// 获取DNS记录列表
	dnsRecords, err := m.Client.GetDnsRecordList(&m.Ibm.DomainBinding.Domain, &m.Ibm.DomainBinding.SubDomain)
	if err != nil {
		m.Log.Errorf("获取DNS记录失败: %v", err)
	}

	// 删除无效DNS记录
	for _, record := range dnsRecords {
		if record.PublicIp != nil && currentIPs[*record.PublicIp] == "" {
			m.Log.Infof("删除无效DNS记录: %s", *record.PublicIp)
			if err := m.Client.RemoveDNSRecord(&m.Ibm.DomainBinding.Domain, record.RecordId); err != nil {
				m.Log.Errorf("删除DNS记录失败: %v", err)
			}
		}
	}

	// 添加新DNS记录
	validRecords := 0
	for _, record := range dnsRecords {
		if record.PublicIp != nil && currentIPs[*record.PublicIp] != "" {
			validRecords++
		}
	}
	if validRecords < m.Ibm.DomainBinding.PraseNum {
		needAdd := m.Ibm.DomainBinding.PraseNum - validRecords
		m.Log.Infof("可以添加 %d 条DNS记录", needAdd)

		added := 0
		for ip := range currentIPs {
			if added >= needAdd {
				break
			}

			// 检查是否已存在该IP的记录
			exists := false
			for _, record := range dnsRecords {
				if record.PublicIp != nil && *record.PublicIp == ip {
					exists = true
					break
				}
			}

			if !exists {
				ipCopy := ip
				m.Log.Infof("添加DNS记录: %s", ip)
				if err := m.Client.AddDNSRecord(&tcloud.DnsRecordP{
					Domain:     &m.Ibm.DomainBinding.Domain,
					SubDomain:  &m.Ibm.DomainBinding.SubDomain,
					RecordType: &m.Ibm.DomainBinding.RecordType,
					RecordLine: &m.Ibm.DomainBinding.RecordLine,
					Value:      &ipCopy,
					TTL:        &m.Ibm.DomainBinding.TTL,
				}); err != nil {
					m.Log.Errorf("添加DNS记录失败: %v", err)
				} else {
					added++
				}
			}
		}
	}

	return currentIPs, nil
}

// 初始化实例（根据配置上传文件并执行命令）
func (m *InstanceManager) InitIns(ips map[string]string, a *tcloud.AClient, ibm *utils.InstanceBindingManager, region, uin string) error {
	rows, err := a.GetTag(m.Cfg.Other["execFlagTagKey"].(string), "true")
	tagIns := make(map[string]bool, 0)
	if err == nil {
		for _, res := range rows {
			if *res.ServiceType == "cvm" && *res.ResourcePrefix == "instance" && *res.ResourceRegion == region {
				tagIns[*res.ResourceId] = true
			}
		}
	}

	for ip, insId := range ips {
		if !tagIns[insId] {
			// m.Log, ibm.Feature.CommandExec.LogFile.
			ssh, err := utils.NewSClient(ip, 22, ibm.Instance.UserConfig.Username, ibm.Instance.UserConfig.Password, m.Log)
			if err != nil {
				return err
			}

			defer ssh.Close()
			if ibm.Feature.FileTransfer.Enabled {
				m.Log.Info("开始上传文件")
				err = ssh.Upload(ibm.Feature.FileTransfer.LocalPath, ibm.Feature.FileTransfer.RemotePath)
				if err != nil {
					return err
				}
			}

			if ibm.Feature.CommandExec.Enabled {
				m.Log.Info("开始执行命令")
				_, err = ssh.ExecCommand(ibm.Feature.CommandExec.Command)
				if err != nil {
					return err
				}
			}

			err = a.AddTag(m.Cfg.Other["execFlagTagKey"].(string), "true", region, uin, insId)
			if err != nil {
				m.Log.Errorf("添加标签失败: %v", err)
			}
		} else {
			m.Log.Infof("实例 %s 已执行命令，跳过", insId)
		}

	}

	return nil
}

// checkIns 检查并维护实例数量到期望状态
func (m *InstanceManager) checkIns() {
	fields := logrus.Fields{
		"实例管理器": m.Ibm.Name,
		"区域":    m.Region,
		"可用区":   m.Zone,
	}
	m.Log.WithFields(fields).Debug("开始检查实例状态")

	// 获取当前实例数量
	desiredCount := m.Ibm.AutoMaintenance.DesiredCount
	currentCount, err := m.Client.GetInstanceCount(m.Cfg.TConfig.TagKey, m.Ibm.Name)
	if err != nil {
		m.Log.WithFields(fields).Errorf("获取实例数量失败: %v", err)
		return
	}

	m.Log.WithFields(logrus.Fields{
		"当前实例数量": currentCount,
		"指定实例数量": desiredCount,
	}).Info("实例数量检查")

	switch {
	case currentCount < desiredCount:
		// 实例不足，创建新实例
		m.Log.WithField("count", desiredCount-currentCount).Info("需要创建新实例")
		m.InsCfg.InstanceCount = desiredCount - currentCount
		if _, err := m.Client.RunInstances(m.InsCfg); err != nil {
			m.Log.WithFields(fields).Errorf("创建实例失败: %v", err)
			return
		}

		// 同步DNS记录
		if ips, err := m.syncDNSRecords(); err != nil {
			m.Log.WithFields(fields).Errorf("同步DNS记录失败: %v", err)
		} else if err := m.InitIns(ips, m.Client, m.Ibm, m.Region, m.Cfg.Uin); err != nil {
			m.Log.WithFields(fields).Errorf("初始化实例失败: %v", err)
		} else {
			m.Log.Info("实例创建和初始化完成")
		}

	case currentCount > desiredCount && m.Ibm.AutoMaintenance.AutoRemove:
		// 实例过多，删除多余实例
		removeCount := currentCount - desiredCount
		m.Log.WithField("count", removeCount).Info("删除多余实例")
		if err := m.Client.RandomDelete(int64(removeCount)); err != nil {
			m.Log.WithFields(fields).Errorf("删除实例失败: %v", err)
		}

	default:
		// 实例数量正常，检查初始化状态
		if m.Ibm.Feature.FileTransfer.Enabled || m.Ibm.Feature.CommandExec.Enabled {
			m.Log.Debug("检查实例初始化状态")
			if ips, err := m.syncDNSRecords(); err != nil {
				m.Log.WithFields(fields).Errorf("同步DNS记录失败: %v", err)
			} else if err := m.InitIns(ips, m.Client, m.Ibm, m.Region, m.Cfg.Uin); err != nil {
				m.Log.WithFields(fields).Errorf("初始化实例失败: %v", err)
			} else {
				m.Log.Debug("实例初始化检查完成")
			}
		}
	}
}
