package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type SClient struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	log        *logrus.Logger
}

// NewSftpClient 创建SFTP客户端
func NewSClient(host string, port int, username, password string, log *logrus.Logger) (*SClient, error) {
	// 创建SSH配置
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// 连接SSH服务器
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("SFTP客户端创建失败: %v", err)
	}

	client := &SClient{
		sshClient:  sshClient,
		sftpClient: sftpClient,
		log:        log,
	}
	return client, nil
}

// Close 关闭连接
func (c *SClient) Close() error {
	c.log.Info("正在关闭SSH/SFTP连接")
	if c.sftpClient != nil {
		c.sftpClient.Close()
	}
	if c.sshClient != nil {
		err := c.sshClient.Close()
		if err != nil {
			c.log.Errorf("关闭SSH连接失败: %v", err)
			return err
		}
	}
	c.log.Info("SSH/SFTP连接已关闭")
	return nil
}

// Upload 上传文件或文件夹
func (c *SClient) Upload(localPath, remotePath string) error {
	c.log.WithFields(logrus.Fields{
		"localPath":  localPath,
		"remotePath": remotePath,
	}).Info("开始上传文件/目录")

	// 获取本地文件信息
	localInfo, err := os.Stat(localPath)
	if err != nil {
		c.log.Errorf("获取本地路径信息失败: %v", err)
		return fmt.Errorf("获取本地路径信息失败: %v", err)
	}

	// 确保远程目录存在
	if err := c.ensureRemoteDir(remotePath); err != nil {
		return err
	}

	// 如果是目录，上传整个目录
	if localInfo.IsDir() {
		return c.uploadDir(localPath, remotePath)
	}

	// 如果是文件，上传单个文件
	return c.uploadFile(localPath, remotePath+"/"+filepath.Base(localPath))
}

// uploadFile 上传单个文件
func (c *SClient) uploadFile(localFilePath, remoteFilePath string) error {
	// 打开本地文件
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %v", err)
	}
	defer localFile.Close()
	// 创建远程文件
	remoteFile, err := c.sftpClient.Create(remoteFilePath)
	c.sftpClient.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %v", err)
	}
	defer remoteFile.Close()
	c.log.WithFields(logrus.Fields{
		"local":  localFilePath,
		"remote": remoteFilePath,
	}).Debug("文件正在上传...")
	// 复制文件内容
	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("上传文件内容失败: %v", err)
	}

	return nil
}

// uploadDir 上传整个目录
func (c *SClient) uploadDir(localDirPath, remoteDirPath string) error {
	// 遍历本地目录
	err := filepath.Walk(localDirPath, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localDirPath, localPath)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %v", err)
		}

		// 转换Windows路径分隔符为Linux格式
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		// 构建远程路径
		remotePath := path.Join(remoteDirPath, relPath)

		// 如果是目录，确保远程目录存在
		if info.IsDir() {
			return c.ensureRemoteDir(remotePath)
		}

		// 上传文件
		return c.uploadFile(localPath, remotePath)
	})

	if err != nil {
		return err
	}
	return nil
}

// Download 下载文件或文件夹
func (c *SClient) Download(remotePath, localPath string) error {
	// 获取远程文件信息
	remoteInfo, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("获取远程路径信息失败: %v", err)
	}

	// 如果是目录，下载整个目录
	if remoteInfo.IsDir() {
		return c.downloadDir(remotePath, localPath)
	}

	// 如果是文件，下载单个文件
	return c.downloadFile(remotePath, localPath)
}

// downloadFile 下载单个文件
func (c *SClient) downloadFile(remoteFilePath, localFilePath string) error {
	// 确保本地目录存在
	localDir := filepath.Dir(localFilePath)
	if err := os.MkdirAll(localDir, os.ModePerm); err != nil {
		return fmt.Errorf("创建本地目录失败: %v", err)
	}

	// 打开远程文件
	remoteFile, err := c.sftpClient.Open(remoteFilePath)
	if err != nil {
		return fmt.Errorf("打开远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	// 创建本地文件
	localFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer localFile.Close()

	// 复制文件内容
	if _, err := io.Copy(localFile, remoteFile); err != nil {
		return fmt.Errorf("下载文件内容失败: %v", err)
	}

	return nil
}

// downloadDir 下载整个目录
func (c *SClient) downloadDir(remoteDirPath, localDirPath string) error {
	// 确保本地目录存在
	if err := os.MkdirAll(localDirPath, os.ModePerm); err != nil {
		return fmt.Errorf("创建本地目录失败: %v", err)
	}

	// 遍历远程目录
	walker := c.sftpClient.Walk(remoteDirPath)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		// 计算相对路径
		relPath := strings.TrimPrefix(walker.Path(), remoteDirPath)
		if relPath == "" {
			continue
		}

		// 构建本地路径
		localPath := filepath.Join(localDirPath, relPath)

		// 如果是目录，确保本地目录存在
		if walker.Stat().IsDir() {
			if err := os.MkdirAll(localPath, os.ModePerm); err != nil {
				return fmt.Errorf("创建本地目录失败: %v", err)
			}
			continue
		}

		// 下载文件
		if err := c.downloadFile(walker.Path(), localPath); err != nil {
			return err
		}
	}

	return nil
}

// ensureRemoteDir 确保远程目录存在
func (c *SClient) ensureRemoteDir(remoteDir string) error {
	// 检查目录是否存在
	_, err := c.sftpClient.Stat(remoteDir)
	if err == nil {
		return nil
	}

	// 如果不存在，创建目录
	if err := c.sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("创建远程目录失败: %v", err)
	}

	return nil
}

// ExecCommand 执行SSH命令并流式收集日志
func (c *SClient) ExecCommand(command string) (string, error) {
	c.log.WithField("command", command).Info("开始执行SSH命令")

	session, err := c.sshClient.NewSession()
	if err != nil {
		c.log.Errorf("创建SSH会话失败: %v", err)
		return "", fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()

	// 创建管道获取实时输出
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		c.log.Errorf("获取标准输出管道失败: %v", err)
		return "", fmt.Errorf("获取标准输出管道失败: %v", err)
	}

	stderrPipe, err := session.StderrPipe()
	if err != nil {
		c.log.Errorf("获取标准错误管道失败: %v", err)
		return "", fmt.Errorf("获取标准错误管道失败: %v", err)
	}

	// 启动命令
	if err := session.Start(command); err != nil {
		c.log.Errorf("启动命令失败: %v", err)
		return "", fmt.Errorf("启动命令失败: %v", err)
	}

	// 创建缓冲区收集输出
	var outputBuf strings.Builder

	// 实时读取标准输出
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuf.WriteString(line + "\n")
			c.log.WithField("output", line).Info("命令输出")
		}
	}()

	// 实时读取标准错误
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuf.WriteString(line + "\n")
			c.log.WithField("error", line).Warn("命令错误输出")
		}
	}()

	// 等待命令完成
	err = session.Wait()
	output := outputBuf.String()

	if err != nil {
		c.log.WithFields(logrus.Fields{
			"error":  err,
			"output": output,
		}).Error("执行SSH命令失败")
		return output, fmt.Errorf("执行命令失败: %v, 输出: %s", err, output)
	}

	return output, nil
}
