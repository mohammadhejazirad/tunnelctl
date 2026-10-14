package app

import (
	"tunnelctl/internal/domain"
	"tunnelctl/internal/store"
	"tunnelctl/internal/subscription"
)

func (s *Service) Subscriptions() subscription.Manager { return subscription.Manager{Store: s.Store} }
func (s *Service) RenameProfile(key, name string) (domain.PublicProfile, error) {
	name = domain.SafeName(name)
	var out domain.PublicProfile
	if name == "" || len(name) > 256 {
		return out, domain.Fail(2, "Invalid profile name.", "Use a printable name of 1–256 bytes.")
	}
	e := s.Store.Update(func(v *domain.Settings) error {
		p, e := store.Resolve(*v, key)
		if e != nil {
			return e
		}
		for i := range v.Profiles {
			if v.Profiles[i].ID == p.ID {
				v.Profiles[i].Name = name
				v.Profiles[i].UserNamed = true
				out = v.Profiles[i].Public(v.Selected == p.ID)
			}
		}
		return nil
	})
	return out, e
}
func (s *Service) RemoveProfile(key string) error {
	active, e := s.Status()
	if e != nil {
		return e
	}
	return s.Store.Update(func(v *domain.Settings) error {
		p, e := store.Resolve(*v, key)
		if e != nil {
			return e
		}
		if active.ProfileID == p.ID && active.ProcessRunning {
			return domain.Fail(6, "This profile is used by the active session.", "Stop the session before removing its profile.")
		}
		for i := range v.Profiles {
			if v.Profiles[i].ID == p.ID {
				v.Profiles = append(v.Profiles[:i], v.Profiles[i+1:]...)
				break
			}
		}
		if v.Selected == p.ID {
			v.Selected = ""
		}
		return nil
	})
}
