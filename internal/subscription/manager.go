package subscription

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"tunnelctl/internal/domain"
	"tunnelctl/internal/store"
)

type Manager struct {
	Store   *store.Store
	Fetcher Fetcher
}

func resolve(v domain.Settings, key string) (domain.Subscription, error) {
	for _, s := range v.Subscriptions {
		if s.ID == key {
			return s, nil
		}
	}
	for _, s := range v.Subscriptions {
		if s.Name == key {
			return s, nil
		}
	}
	return domain.Subscription{}, domain.Fail(2, "Subscription not found.", "Use subscription list and its full ID or unique name.")
}
func public(v domain.Settings, s domain.Subscription) domain.PublicSubscription {
	n := 0
	for _, p := range v.Profiles {
		if p.SubscriptionID == s.ID {
			n++
		}
	}
	return s.Public(n)
}
func (m Manager) Add(name string, r io.Reader, cidrs []string, caFile string) (domain.PublicSubscription, error) {
	b, e := io.ReadAll(io.LimitReader(r, 16385))
	if e != nil || len(b) > 16384 {
		return domain.PublicSubscription{}, domain.Fail(2, "Cannot read bounded subscription input.", "Provide one HTTPS URL of at most 16 KiB through stdin.")
	}
	id, e := domain.NewID()
	if e != nil {
		return domain.PublicSubscription{}, e
	}
	previewKey, e := domain.NewID()
	if e != nil {
		return domain.PublicSubscription{}, e
	}
	name = domain.SafeName(name)
	if name == "" || len(name) > 256 {
		return domain.PublicSubscription{}, domain.Fail(2, "Invalid subscription name.", "Use a printable name of 1–256 bytes.")
	}
	s := domain.Subscription{ID: id, Name: name, URL: strings.TrimSpace(string(b)), AllowedCIDRs: cidrs, CAFile: caFile, Revision: 1, PreviewKey: previewKey}
	if e = ValidateSource(s); e != nil {
		return domain.PublicSubscription{}, e
	}
	e = m.Store.Update(func(v *domain.Settings) error {
		if len(v.Subscriptions) >= 1000 {
			return domain.Fail(2, "Subscription count limit exceeded.", "Keep at most 1000 subscriptions.")
		}
		for _, old := range v.Subscriptions {
			if old.Name == s.Name || old.URL == s.URL {
				return domain.Fail(6, "This subscription name or source already exists.", "Use subscription list to identify the existing source.")
			}
		}
		v.Subscriptions = append(v.Subscriptions, s)
		return nil
	})
	return s.Public(0), e
}
func (m Manager) List() ([]domain.PublicSubscription, error) {
	v, e := m.Store.Load()
	if e != nil {
		return nil, e
	}
	out := []domain.PublicSubscription{}
	for _, s := range v.Subscriptions {
		out = append(out, public(v, s))
	}
	return out, nil
}
func (m Manager) Show(key string) (domain.PublicSubscription, error) {
	v, e := m.Store.Load()
	if e != nil {
		return domain.PublicSubscription{}, e
	}
	s, e := resolve(v, key)
	return public(v, s), e
}
func (m Manager) Rename(key, name string) (domain.PublicSubscription, error) {
	name = domain.SafeName(name)
	var out domain.PublicSubscription
	e := m.Store.Update(func(v *domain.Settings) error {
		s, e := resolve(*v, key)
		if e != nil {
			return e
		}
		if name == "" || len(name) > 256 {
			return domain.Fail(2, "Invalid subscription name.", "Use 1–256 printable bytes.")
		}
		for _, other := range v.Subscriptions {
			if other.Name == name && other.ID != s.ID {
				return domain.Fail(6, "Subscription name is already in use.", "Choose a unique name.")
			}
		}
		for i := range v.Subscriptions {
			if v.Subscriptions[i].ID == s.ID {
				v.Subscriptions[i].Name = name
				v.Subscriptions[i].Revision++
				out = public(*v, v.Subscriptions[i])
				break
			}
		}
		return nil
	})
	return out, e
}
func (m Manager) Remove(key string) error {
	return m.Store.Update(func(v *domain.Settings) error {
		s, e := resolve(*v, key)
		if e != nil {
			return e
		}
		for i := range v.Profiles {
			if v.Profiles[i].SubscriptionID == s.ID {
				v.Profiles[i].SubscriptionID = ""
			}
		}
		for i, sub := range v.Subscriptions {
			if sub.ID == s.ID {
				v.Subscriptions = append(v.Subscriptions[:i], v.Subscriptions[i+1:]...)
				break
			}
		}
		return nil
	})
}

type Diff struct {
	SubscriptionID   string                 `json:"subscription_id"`
	Revision         uint64                 `json:"revision"`
	Preview          bool                   `json:"preview"`
	NotModified      bool                   `json:"not_modified"`
	Token            string                 `json:"token,omitempty"`
	Added            []domain.PublicProfile `json:"added"`
	Removed          []domain.PublicProfile `json:"removed"`
	RetainedSelected []domain.PublicProfile `json:"retained_selected"`
	Unchanged        int                    `json:"unchanged"`
	Committed        bool                   `json:"committed"`
}

func plan(v domain.Settings, s domain.Subscription, incoming []domain.Profile) (Diff, []domain.Profile) {
	d := Diff{SubscriptionID: s.ID, Revision: s.Revision, Added: []domain.PublicProfile{}, Removed: []domain.PublicProfile{}, RetainedSelected: []domain.PublicProfile{}}
	old := map[string]domain.Profile{}
	for _, p := range v.Profiles {
		if p.SubscriptionID == s.ID {
			old[p.Identity()] = p
		}
	}
	next := []domain.Profile{}
	for _, p := range incoming {
		p.SubscriptionID = s.ID
		if prior, ok := old[p.Identity()]; ok {
			p.ID = prior.ID
			p.CreatedAt = prior.CreatedAt
			p.UserNamed = prior.UserNamed
			p.Name = prior.Name
			p.Tags = append([]string(nil), prior.Tags...)
			p.Favorite = prior.Favorite
			delete(old, p.Identity())
			d.Unchanged++
		} else {
			shown := p.Public(false)
			shown.ID = ""
			d.Added = append(d.Added, shown)
		}
		next = append(next, p)
	}
	for _, p := range v.Profiles {
		if p.SubscriptionID != s.ID {
			continue
		}
		if _, ok := old[p.Identity()]; !ok {
			continue
		}
		if p.ID == v.Selected {
			d.RetainedSelected = append(d.RetainedSelected, p.Public(true))
			p.SubscriptionID = ""
			next = append(next, p)
		} else {
			d.Removed = append(d.Removed, p.Public(false))
		}
	}
	return d, next
}
func (m Manager) Update(ctx context.Context, key string, preview, acceptRemovals bool, expectedToken string) (d Diff, err error) {
	v, e := m.Store.Load()
	if e != nil {
		return d, e
	}
	s, e := resolve(v, key)
	if e != nil {
		return d, e
	}
	lock, e := store.Acquire(filepath.Join(m.Store.Paths.Config, "subscription-"+s.ID+".lock"))
	if e != nil {
		return d, e
	}
	defer func() { err = errors.Join(err, store.Release(lock)) }()
	res, e := m.Fetcher.Fetch(ctx, s)
	if e != nil {
		return d, e
	}
	if res.NotModified {
		if s.UpdatedAt == nil {
			return d, networkError("The server returned 304 before any healthy snapshot existed.")
		}
		return Diff{SubscriptionID: s.ID, Revision: s.Revision, Preview: preview, NotModified: true, Added: []domain.PublicProfile{}, Removed: []domain.PublicProfile{}, RetainedSelected: []domain.PublicProfile{}}, nil
	}
	incoming, e := Parse(res.Body)
	if e != nil {
		return d, e
	}
	d, _ = plan(v, s, incoming)
	keyBytes, e := hex.DecodeString(s.PreviewKey)
	if e != nil || len(keyBytes) != 16 {
		return d, domain.Fail(2, "The private subscription preview key is invalid.", "Restore a healthy private snapshot; no update was committed.")
	}
	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(s.ID + ":" + s.Name + ":" + strconv.FormatUint(s.Revision, 10) + ":"))
	mac.Write(res.Body)
	d.Token = hex.EncodeToString(mac.Sum(nil))
	d.Preview = preview
	if expectedToken != "" && expectedToken != d.Token {
		return d, domain.Fail(6, "The preview is stale.", "Run update --preview again; nothing was committed.")
	}
	if preview {
		return d, nil
	}
	if ctx.Err() != nil {
		return d, networkError("Subscription update was canceled before commit.")
	}
	if len(d.Removed) > 0 && !acceptRemovals {
		return d, domain.Fail(6, "The update would remove subscription profiles.", "Preview the changes and explicitly pass --accept-removals, optionally --if-preview TOKEN.")
	}
	e = m.Store.Update(func(current *domain.Settings) error {
		if ctx.Err() != nil {
			return networkError("Subscription update was canceled before commit.")
		}
		latest, e := resolve(*current, s.ID)
		if e != nil || latest.Revision != s.Revision {
			return domain.Fail(6, "Subscription changed while the update was staged.", "Retry preview; no staged data was committed.")
		}
		fresh, next := plan(*current, latest, incoming)
		if len(fresh.Removed) > 0 && !acceptRemovals {
			return domain.Fail(6, "The staged update now includes removals.", "Preview and explicitly accept removals.")
		}
		profiles := []domain.Profile{}
		for _, p := range current.Profiles {
			if p.SubscriptionID != s.ID {
				profiles = append(profiles, p)
			}
		}
		profiles = append(profiles, next...)
		if len(profiles) > MaxProfiles {
			return parseError()
		}
		current.Profiles = profiles
		now := time.Now().UTC()
		for i := range current.Subscriptions {
			if current.Subscriptions[i].ID == s.ID {
				current.Subscriptions[i].ETag = res.ETag
				current.Subscriptions[i].LastModified = res.LastModified
				current.Subscriptions[i].UpdatedAt = &now
				current.Subscriptions[i].Revision++
			}
		}
		token := d.Token
		d = fresh
		d.Token = token
		d.Committed = true
		return nil
	})
	if e != nil {
		d.Committed = false
		return d, e
	}
	return d, nil
}
