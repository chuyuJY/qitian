package main

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

var (
	ascii = `
___ _ _  _ _   _ ___  ____ _    ____ _  _ ____ ____ ____ 
 |  | |\ |  \_/  |__] |__| |    |__| |\ | |    |___ |__/ 
 |  | | \|   |   |__] |  | |___ |  | | \| |___ |___ |  \                                        
`
)

type Config struct {
	SSLCertificateKey   string      `yaml:"ssl_certificate_key"`   // https时用的密钥
	Location            []*Location `yaml:"location"`              // 均衡器细节
	Schema              string      `yaml:"schema"`                // http 或 https
	Port                int         `yaml:"port"`                  // 端口号
	SSLCertificate      string      `yaml:"ssl_certificate"`       // https时需要的证书
	HealthCheck         bool        `yaml:"tcp_health_check"`      // 是否开启健康检测
	HealthCheckInterval uint        `yaml:"health_check_interval"` // 健康检测间隔
	MaxAllowed          uint        `yaml:"max_allowed"`
}

// Location 负载均衡的路由细节
type Location struct {
	Pattern     string   `yaml:"pattern"`      // 代理的路由
	ProxyPass   []string `yaml:"proxy_pass"`   // 代理主机集
	BalanceMode string   `yaml:"balance_mode"` // 负载均衡算法
}

func ReadConfig(fileName string) (*Config, error) {
	in, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(in, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) Validation() error {
	if c.Schema != "http" && c.Schema != "https" {
		return fmt.Errorf("the schema \"%s\" not supported", c.Schema)
	}
	if len(c.Location) == 0 {
		return errors.New("the details of location cannot be null")
	}
	if c.Schema == "https" && (len(c.SSLCertificate) == 0 || len(c.SSLCertificateKey) == 0) {
		return errors.New("the https proxy requires ssl_certificate_key and ssl_certificate")
	}
	if c.HealthCheckInterval < 1 {
		return errors.New("health_check_interval must be greater than 0")
	}
	return nil
}

// Print print config details
func (c *Config) Print() {
	fmt.Printf("%s\nSchema: %s\nPort: %d\nHealth Check: %v\nLocation:\n",
		ascii, c.Schema, c.Port, c.HealthCheck)
	for _, l := range c.Location {
		fmt.Printf("\tRoute: %s\n\tProxy Pass: %s\n\tMode: %s\n\n",
			l.Pattern, l.ProxyPass, l.BalanceMode)
	}
}
