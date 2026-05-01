package domain

import (
	"fmt"
	"net/url"
	"strings"
)

func RepoIDFromRemoteURL(remoteURL string) (string, error) {
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(remoteURL, "git@"), ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid ssh repository url: %s", remoteURL)
		}
		host := strings.TrimSpace(parts[0])
		path := normalizeRemotePath(parts[1])
		if host == "" || path == "" {
			return "", fmt.Errorf("invalid ssh repository url: %s", remoteURL)
		}
		return host + "/" + path, nil
	}

	u, err := url.Parse(remoteURL)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid repository url: %s", remoteURL)
	}

	path := normalizeRemotePath(strings.TrimPrefix(u.Path, "/"))
	if strings.EqualFold(u.Host, "dev.azure.com") {
		path = strings.Replace(path, "/_git/", "/", 1)
	}
	if path == "" {
		return "", fmt.Errorf("invalid repository url: %s", remoteURL)
	}

	return u.Host + "/" + path, nil
}

func normalizeRemotePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	return path
}
