package config

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
}

type LocalStorageConfig struct {
	BaseDir string
}

type StorageConfig struct {
	Driver string
	S3     S3Config
	Local  LocalStorageConfig
}
