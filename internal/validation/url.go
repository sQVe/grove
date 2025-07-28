package validation

import (
	"net/url"
	"strings"

	"github.com/sqve/grove/internal/errors"
)

var (
	// supportedHostsMap provides O(1) lookup for supported Git platforms.
	supportedHostsMap = map[string]bool{
		"github.com":    true,
		"gitlab.com":    true,
		"bitbucket.org": true,
		"dev.azure.com": true,
		"codeberg.org":  true,
		"gitea.io":      true,
	}

	// supportedHosts provides the list for error messages.
	supportedHosts = []string{
		"github.com", "gitlab.com", "bitbucket.org",
		"dev.azure.com", "codeberg.org", "gitea.io",
	}
)

// ValidateURL validates that a URL is properly formatted and from a supported platform.
func ValidateURL(input string) error {
	if !IsURL(input) {
		return nil // Not a URL, validation handled elsewhere.
	}

	parsed, err := url.Parse(input)
	if err != nil {
		return errors.ErrURLParsing(input, err)
	}

	if parsed.Host == "" {
		return errors.ErrInvalidURL(input, "missing host")
	}

	// Check if the host is supported using exact matching to prevent subdomain attacks.
	hostSupported := supportedHostsMap[parsed.Host]
	if !hostSupported {
		// Also check for subdomains of supported hosts (e.g., api.github.com).
		for host := range supportedHostsMap {
			if strings.HasSuffix(parsed.Host, "."+host) {
				hostSupported = true
				break
			}
		}
	}

	if !hostSupported {
		return errors.ErrUnsupportedURL(input).
			WithContext("supported_platforms", strings.Join(supportedHosts, ", "))
	}

	return nil
}
