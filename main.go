package main

import (
	"context"
	"cvmspot/cli"
	"cvmspot/service"
	"cvmspot/tcloud"
	"cvmspot/utils"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logrus.New()

func initConfig(cfg *utils.Config) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("不能读取配置文件: %v (检查 config.yaml 是否存在或是否 YAML 格式)", err)
	}

	if err := viper.UnmarshalKey("instance_managers", &cfg.IBManager); err != nil {
		return fmt.Errorf("不能初始化配置-实例管理器: %v", err)
	}

	if err := viper.UnmarshalKey("log", &cfg.LogConfig); err != nil {
		return fmt.Errorf("不能初始化配置-日志管理器: %v", err)
	}

	if err := viper.UnmarshalKey("tencentcloud", &cfg.TConfig); err != nil {
		return fmt.Errorf("不能初始化配置-腾讯云管理器: %v", err)
	}

	secretId := strings.TrimSpace(os.Getenv("TENCENTCLOUD_SECRET_ID"))
	secretKey := strings.TrimSpace(os.Getenv("TENCENTCLOUD_SECRET_KEY"))

	// 优先从环境变量获取
	if secretId != "" && secretKey != "" {
		cfg.TConfig.SecretId = secretId
		cfg.TConfig.SecretKey = secretKey
	}

	// 检查必要的配置项
	// requiredKeys := []string{
	// 	"tencentcloud.region",
	// 	"log.level",
	// 	"log.file",
	// }

	// for _, key := range requiredKeys {
	// 	if !viper.IsSet(key) {
	// 		return fmt.Errorf("missing required config key: %s", key)
	// 	}
	// }

	return nil
}

func initLogger(logConfig utils.LogConfig) {
	// 设置中文日志格式
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		FieldMap: logrus.FieldMap{
			"time": "Date",
			// "level": "Level",
			// "msg":   "Msg",
		},
	})

	level, err := logrus.ParseLevel(logConfig.Level)
	if err != nil {
		level = logrus.InfoLevel
		log.Warnf("不能格式化日志等级 '%s', 默认设置为 INFO 级别", logConfig.Level)
	}
	log.SetLevel(level)

	// 设置日志输出
	var writers []io.Writer
	writers = append(writers, os.Stdout) // 总是输出到控制台

	if logConfig.LogPath != "" {
		file, err := os.OpenFile(logConfig.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Warnf("不能访问日志文件: %v, 只打印控制台", err)
		} else {
			writers = append(writers, file)
		}
	}

	log.SetOutput(io.MultiWriter(writers...))
}

func main() {

	var cfg utils.Config
	// 初始化配置
	if err := initConfig(&cfg); err != nil {
		log.Fatal(err)
	}
	cfg.IsCli = len(os.Args) > 1

	// 初始化日志
	initLogger(cfg.LogConfig)

	// 初始化腾讯云客户端
	client, err := tcloud.NewClientWithLogger(
		cfg,
		log,
	)

	if err != nil {
		panic("腾讯云客户端初始化失败..")
	}

	if cfg.IsCli {
		// CLI模式
		cli.Execute(client)
	} else {
		// 服务模式
		log.Info("服务模式启动中...")
		go startService(&cfg, client)

		// 等待终止信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info("已退出服务模式...")
	}
}

func startService(cfg *utils.Config, client *tcloud.Client) {
	// 初始化配置参数
	cfg.SetConfig()

	// 创建实例管理器组
	//managerGroup := service.NewInstanceManagerGroup(log, client)
	managerGroup := service.NewInstanceManagerGroup(client, cfg)

	// 启动实例管理循环
	managerGroup.Run(context.Background())

}
