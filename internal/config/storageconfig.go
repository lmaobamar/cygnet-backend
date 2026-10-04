package config

import "fmt"

type DriverType string

const (
	DriverS3    DriverType = "s3"
	DriverLocal DriverType = "local"
)

type S3Config struct {
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string `json:",omitempty"`
	UsePathStyle    bool   `json:",omitempty"`
	BaseURL         string
}

type LocalStorageConfig struct {
	BaseDir string
}

type StorageConfig struct {
	Driver string
	S3     S3Config
	Local  LocalStorageConfig
}

func (s StorageConfig) Validate() error {
	switch s.Driver {
	case string(DriverS3):
		if s.S3.Bucket == "" || s.S3.AccessKeyID == "" || s.S3.SecretAccessKey == "" {
			return fmt.Errorf("DriverS3 requires Bucket, AccessKeyID, and SecretAccessKey")
		}
		if s.S3.BaseURL == "" {
			return fmt.Errorf("DriverS3 requires BaseURL. For example https://cdn.example.com")
		}
	case string(DriverLocal):
		if s.Local.BaseDir == "" {
			return fmt.Errorf("DriverLocal requires non-empty BaseDir")
		}
	case "":
		return fmt.Errorf("Storage: a Driver is required")
	default:
		return fmt.Errorf("Invalid Storage driver")
	}
	return nil
}
