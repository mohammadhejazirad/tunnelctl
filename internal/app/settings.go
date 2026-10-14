package app

import (
	"time"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/store"
)

type PreferencesSnapshot struct {
	Revision    uint64             `json:"revision"`
	Preferences domain.Preferences `json:"preferences"`
}

func (s *Service) Settings() (PreferencesSnapshot, error) {
	v, e := s.Store.Load()
	if e != nil {
		return PreferencesSnapshot{}, e
	}
	return PreferencesSnapshot{v.Revision, v.Preferences}, nil
}

// EditSettings reads under the shared writer lock. A form must supply the revision
// it opened with; field patches may omit it because they edit the latest snapshot.
func (s *Service) EditSettings(revision *uint64, edit func(*domain.Preferences) error) (PreferencesSnapshot, error) {
	var result PreferencesSnapshot
	e := s.Store.Update(func(v *domain.Settings) error {
		if revision != nil && *revision != v.Revision {
			return domain.Fail(6, "Settings changed while this form was open.", "Reload settings and reapply your changes; nothing was overwritten.")
		}
		if e := edit(&v.Preferences); e != nil {
			return e
		}
		if e := v.Preferences.Validate(); e != nil {
			return e
		}
		result = PreferencesSnapshot{v.Revision + 1, v.Preferences}
		return nil
	})
	if e != nil {
		return PreferencesSnapshot{}, e
	}
	return result, nil
}

func (s *Service) SaveSettings(revision uint64, p domain.Preferences) (PreferencesSnapshot, error) {
	return s.EditSettings(&revision, func(v *domain.Preferences) error { *v = p; return nil })
}

// WithPreferences changes defaults for subsequent operations only. Existing
// supervisors retain their captured executable, listener, credentials and deadline.
func (s Service) WithPreferences(p domain.Preferences, coreOverride, timeoutOverride bool) *Service {
	if !coreOverride {
		s.CorePath = DefaultCore()
		if p.CorePath != "" {
			s.CorePath = p.CorePath
		}
	}
	if !timeoutOverride {
		s.Timeout, _ = time.ParseDuration(p.Timeout)
	}
	return &s
}

func (s *Service) Paths() store.Paths { return s.Store.Paths }

func (s *Service) ProfileSettings(key string) (domain.PublicProfile, uint64, error) {
	v, e := s.Store.Load()
	if e != nil {
		return domain.PublicProfile{}, 0, e
	}
	p, e := store.Resolve(v, key)
	if e != nil {
		return domain.PublicProfile{}, 0, e
	}
	return p.Public(p.ID == v.Selected), v.Revision, nil
}

func (s *Service) ProfileMetadata(key string, tags []string, favorite bool, revision *uint64) (domain.PublicProfile, error) {
	if e := domain.ValidateTags(tags); e != nil {
		return domain.PublicProfile{}, e
	}
	var out domain.PublicProfile
	e := s.Store.Update(func(v *domain.Settings) error {
		if revision != nil && *revision != v.Revision {
			return domain.Fail(6, "Profile data changed while editing.", "Reload the form and reapply your changes.")
		}
		p, e := store.Resolve(*v, key)
		if e != nil {
			return e
		}
		for i := range v.Profiles {
			if v.Profiles[i].ID == p.ID {
				v.Profiles[i].Tags = append([]string(nil), tags...)
				v.Profiles[i].Favorite = favorite
				out = v.Profiles[i].Public(v.Selected == p.ID)
			}
		}
		return nil
	})
	return out, e
}
