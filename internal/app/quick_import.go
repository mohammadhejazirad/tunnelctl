package app

import (
	"net/url"
	"strconv"
	"strings"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/importer"
	"tunnelctl/internal/subscription"
)

type QuickImportResult struct {
	Kind    string `json:"kind"`
	Added   int    `json:"added"`
	Skipped int    `json:"skipped,omitempty"`
	Name    string `json:"name,omitempty"`
}

func (s *Service) QuickImport(value string) (QuickImportResult, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return QuickImportResult{}, domain.Fail(2, "Clipboard content is empty.", "Copy a share link or HTTPS subscription URL and retry.")
	}
	if !strings.ContainsAny(value, "\r\n\t") {
		if u, e := url.ParseRequestURI(value); e == nil && strings.EqualFold(u.Scheme, "https") && u.Host != "" {
			name, e := s.subscriptionName(u.Hostname())
			if e != nil {
				return QuickImportResult{}, e
			}
			source, e := s.Subscriptions().Add(name, strings.NewReader(value), nil, "")
			if e != nil {
				return QuickImportResult{}, e
			}
			return QuickImportResult{Kind: "subscription", Added: 1, Name: source.Name}, nil
		}
		if _, e := importer.Parse(strings.NewReader(value)); e == nil {
			profile, e := s.Import(strings.NewReader(value))
			if e != nil {
				return QuickImportResult{}, e
			}
			return QuickImportResult{Kind: "profile", Added: 1, Name: profile.Name}, nil
		}
	}
	profiles, e := subscription.Parse([]byte(value))
	if e != nil {
		return QuickImportResult{}, e
	}
	result := QuickImportResult{Kind: "profiles"}
	e = s.Store.Update(func(settings *domain.Settings) error {
		existing := map[string]bool{}
		for _, profile := range settings.Profiles {
			existing[profile.Identity()] = true
		}
		for _, profile := range profiles {
			if existing[profile.Identity()] {
				result.Skipped++
				continue
			}
			if len(settings.Profiles) >= 10000 {
				return domain.Fail(2, "Profile count limit exceeded.", "Use a separate private profile store.")
			}
			settings.Profiles = append(settings.Profiles, profile)
			existing[profile.Identity()] = true
			result.Added++
			if settings.Selected == "" {
				settings.Selected = profile.ID
			}
		}
		if result.Added == 0 {
			return domain.Fail(6, "All clipboard profiles are already stored.", "Open Profiles to select or manage the existing entries.")
		}
		return nil
	})
	return result, e
}

func (s *Service) subscriptionName(host string) (string, error) {
	base := domain.SafeName(host)
	if base == "" {
		base = "Subscription"
	}
	sources, e := s.Subscriptions().List()
	if e != nil {
		return "", e
	}
	used := map[string]bool{}
	for _, source := range sources {
		used[source.Name] = true
	}
	if !used[base] {
		return base, nil
	}
	for i := 2; i <= 1000; i++ {
		candidate := base + " " + strconv.Itoa(i)
		if !used[candidate] {
			return candidate, nil
		}
	}
	return "", domain.Fail(2, "Cannot choose a subscription name.", "Rename or remove duplicate subscriptions and retry.")
}
