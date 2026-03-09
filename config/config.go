/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package config

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
)

type AccessManagerConfig struct {
	Port             int    `required:"false" default:"4000"`
	MetricServerPort int    `required:"false" split_words:"true" default:"2114"`
	GatewayPort      int    `split_words:"true" default:"8080"`
	MetadataUrls     string `required:"false" default:"localhost"` // a list of hosts or fully fledged URLs
	ServerKey        string `required:"false" default:"./host.key"`
}

func ReadAccessManagerConfig() (*AccessManagerConfig, error) {
	var accessManager AccessManagerConfig

	if err := envconfig.Process("AM_", &accessManager); err != nil {
		return nil, fmt.Errorf("missing accessManager config: %w", err)
	}

	return &accessManager, nil
}
