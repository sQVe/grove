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

	hostSupported := supportedHostsMap[parsed.Host]
	if !hostSupported {
		// Check for valid subdomains (e.g., api.github.com).
		hostSupported = isValidSubdomain(parsed.Host, supportedHostsMap)
	}

	if !hostSupported {
		return errors.ErrUnsupportedURL(input).
			WithContext("supported_platforms", strings.Join(supportedHosts, ", "))
	}

	return nil
}

// Prevents subdomain spoofing attacks like evil.github.com.badsite.com by validating domain hierarchy.
func isValidSubdomain(host string, supportedHosts map[string]bool) bool {
	hostParts := strings.Split(host, ".")
	if len(hostParts) < 2 {
		return false
	}

	for supportedHost := range supportedHosts {
		supportedParts := strings.Split(supportedHost, ".")

		if len(hostParts) <= len(supportedParts) {
			continue
		}

		// Validate suffix matches exactly (e.g., "api.github.com" ends with "github.com").
		hostSuffix := hostParts[len(hostParts)-len(supportedParts):]
		isMatch := true
		for i, part := range supportedParts {
			if hostSuffix[i] != part {
				isMatch = false
				break
			}
		}

		if isMatch {
			// Reject malformed subdomains (empty parts, leading/trailing hyphens).
			subdomainParts := hostParts[:len(hostParts)-len(supportedParts)]
			for _, part := range subdomainParts {
				if part == "" || strings.HasPrefix(part, "-") || strings.HasSuffix(part, "-") {
					return false
				}
			}
			return true
		}
	}

	return false
}
