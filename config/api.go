package config

import (
	"net/url"

	"github.com/mevansam/goutils/logger"
)


func GetApiEndpointNames() []string {

	var (
		err error

		url *url.URL
	)
	
	endpointNames := []string{}
	if url, err = url.Parse(AWS_USERSPACE_API_URL); err != nil {
		logger.ErrorMessage(
			"GetApiEndpointNames(): Error parsing URL '%s': %s", 
			AWS_USERSPACE_API_URL, err.Error(),
		)
	}
	endpointNames = append(endpointNames, url.Hostname())
	return endpointNames
}
