package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Config struct {
	SFTPHost       string `json:"sftp_host"`
	SFTPUser       string `json:"sftp_user"`
	SFTPPassword   string `json:"sftp_password"`
	SFTPDirectory  string `json:"sftp_directory"`
	LocalDirectory string `json:"local_directory"`
}

func loadConfig(filename string) (Config, error) {
	var config Config
	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	return config, err
}

func connectSFTP(config Config) (*sftp.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: config.SFTPUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.SFTPPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", config.SFTPHost, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SFTP server: %w", err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return client, nil
}

func downloadFile(client *sftp.Client, remotePath, localPath string) error {
	srcFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer srcFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to creatae local file: %w", err)
	}
	defer localFile.Close()

	_, err = srcFile.WriteTo(localFile)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

func main() {
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed config loading: %v", err)
	}

	client, err := connectSFTP(config)
	if err != nil {
		log.Fatalf("SFTP connection failure: %v", err)
	}
	defer client.Close()

	files, err := client.ReadDir(config.SFTPDirectory)
	if err != nil {
		log.Fatalf("Read directory %s failed: %v", config.SFTPDirectory, err)
	}

	err = os.MkdirAll(config.LocalDirectory, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create local directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			remotePath := path.Join(config.SFTPDirectory, file.Name())
			localPath := filepath.Join(config.LocalDirectory, file.Name())

			fmt.Printf("Downloading %s -> %s\n", remotePath, localPath)
			err = downloadFile(client, remotePath, localPath)
			if err != nil {
				log.Printf("Failed to download %s: %v", remotePath, err)
			}
		}
	}

	fmt.Println("Download completed")
}
