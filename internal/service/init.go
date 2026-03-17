package service

import "github.com/Albert-Przybyla/swaggergo/internal/config"

func InitProject() error {
	return config.WriteDefault(".swaggergo.yaml")
}
