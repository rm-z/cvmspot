package utils

type InstanceConfig struct {
	InstanceName       string              `mapstructure:"instance_name"`
	Regions            []string            `mapstructure:"regions"`
	ImageId            string              `mapstructure:"image_id"`
	InstanceType       string              `mapstructure:"instance_type"`
	InternetChargeType string              `mapstructure:"internet_charge_type"`
	SystemDisk         SystemDisk          `mapstructure:"system_disk"`
	Internet           Internet            `mapstructure:"internet"`
	VpcConfig          VpcConfig           `mapstructure:"vpc"`
	SubnetConfig       SubnetConfig        `mapstructure:"subnet"`
	SecurityGroups     SecurityGroupConfig `mapstructure:"security_groups"`
	Tags               map[string]string   `mapstructure:"tags"`
	UserConfig         UserConfig          `mapstructure:"user"`
}

type SystemDisk struct {
	Type string `mapstructure:"type"`
	Size int64  `mapstructure:"size"`
}

type Internet struct {
	ChargeType   string `mapstructure:"charge_type"`
	BandwidthOut int64  `mapstructure:"bandwidth_out"`
}

type VpcConfig struct {
	TagVal    string `mapstructure:"tag_val"`
	VpcId     string `mapstructure:"vpc_id"`
	VpcName   string `mapstructure:"vpc_name"`
	CidrBlock string `mapstructure:"cidr_block"`
}

type SubnetConfig struct {
	TagVal     string `mapstructure:"tag_val"`
	SubnetId   string `mapstructure:"subnet_id"`
	SubnetName string `mapstructure:"subnet_name"`
	CidrBlock  string `mapstructure:"cidr_block"`
}

type SecurityGroupConfig struct {
	SecurityGroupId  string       `mapstructure:"security_group_id"`
	SecurityName     string       `mapstructure:"security_name"`
	GroupDescription string       `mapstructure:"group_description"`
	TagVal           string       `mapstructure:"tag_val"`
	Rules            []RuleConfig `mapstructure:"rules"`
}

type RuleConfig struct {
	Type        string `mapstructure:"type"`
	Protocol    string `mapstructure:"protocol"`
	Port        string `mapstructure:"port"`
	CidrIp      string `mapstructure:"cidr_ip"`
	Action      string `mapstructure:"action"`
	Description string `mapstructure:"desc"`
}

type FeatureConfig struct {
	FileTransfer struct {
		LocalPath  string `mapstructure:"local_path"`
		RemotePath string `mapstructure:"remote_path"`
		Enabled    bool   `mapstructure:"enabled"`
	} `mapstructure:"file_transfer"`
	CommandExec struct {
		Enabled bool   `mapstructure:"enabled"`
		Command string `mapstructure:"command"`
	} `mapstructure:"command_exec"`
}

type DomainBindingConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	TagKey     string `mapstructure:"tag_key"`
	Domain     string `mapstructure:"domain"`
	SubDomain  string `mapstructure:"subdomain"`
	RecordLine string `mapstructure:"record_line"`
	RecordType string `mapstructure:"record_type"`
	PraseNum   int    `mapstructure:"prase_num"`
	TTL        uint64 `mapstructure:"ttl"`
}

type AutoMaintenanceConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	CheckInterval int64  `mapstructure:"check_interval"`
	DesiredCount  int64  `mapstructure:"desired_count"`
	LowestPrice   string `mapstructure:"lowest_price"`
	AutoRemove    bool   `mapstructure:"auto_remove"`
}

type InstanceBindingManager struct {
	Name            string                `mapstructure:"name"`
	Instance        InstanceConfig        `mapstructure:"instance"`
	Feature         FeatureConfig         `mapstructure:"feature"`
	DomainBinding   DomainBindingConfig   `mapstructure:"domain_binding"`
	AutoMaintenance AutoMaintenanceConfig `mapstructure:"auto_maintenance"`
}

type LogConfig struct {
	LogPath string `mapstructure:"log_path"`
	Level   string `mapstructure:"level"`
}

type TConfig struct {
	SecretId  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
	TagKey    string `mapstructure:"tag_key"`
}

type Config struct {
	TConfig   TConfig
	IBManager []InstanceBindingManager
	LogConfig LogConfig
	IsCli     bool
	Uin       string
	Other     map[string]interface{}
}

type UserConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (cfg *Config) SetConfig() {
	cfg.Other = make(map[string]interface{})
	cfg.Other["execFlagTagKey"] = "exec"
	cfg.Other["execFlagTagVal"] = "true"
}
