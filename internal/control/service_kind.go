package control

import (
	"errors"
	"strings"
)

const (
	ServiceUninstalled   = "uninstalled"
	ServiceAuthorization = "authorization"
	ServiceStopped       = "stopped"
	ServiceUnreachable   = "unreachable"
)

// IsAuthorization reports a rejected desktop peer.
func IsAuthorization(err error) bool {
	return err != nil && (errors.Is(err, ErrAuthorization) || strings.Contains(err.Error(), ErrAuthorization.Error()))
}

// ServiceKind classifies a failed dial for the desktop. An empty string means the service answered.
func ServiceKind(installed bool, err error) string {
	if err == nil {
		return ""
	}
	if IsAuthorization(err) {
		return ServiceAuthorization
	}
	if IsPipeAbsent(err) || IsEndpointDown(err) {
		if installed {
			return ServiceStopped
		}
		return ServiceUninstalled
	}
	return ServiceUnreachable
}
