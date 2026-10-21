package core

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"picberth/internal/imaging"
)

// Settings are the runtime-editable business settings (docs/api.md §3).
type Settings struct {
	MaxUploadBytes       int64    `json:"maxUploadBytes"`
	MaxPixels            int64    `json:"maxPixels"`
	AllowedFormats       []string `json:"allowedFormats"`
	Dedupe               bool     `json:"dedupe"`
	DefaultVisibility    string   `json:"defaultVisibility"`
	TrashRetentionDays   int      `json:"trashRetentionDays"`
	UploadsPerMinute     int      `json:"uploadsPerMinute"`
	SessionIdleMinutes   int      `json:"sessionIdleMinutes"`
	SessionAbsoluteHours int      `json:"sessionAbsoluteHours"`
}

// SettingsView is Settings plus the read-only infrastructure values.
type SettingsView struct {
	Settings
	PublicURL          string `json:"publicUrl"`
	ImageURL           string `json:"imageUrl"`
	HardMaxUploadBytes int64  `json:"hardMaxUploadBytes"`
}

// SettingsPatch is a partial update. Read-only fields are accepted only when they equal the current value
// (clients may echo a whole GET response back).
type SettingsPatch struct {
	MaxUploadBytes       *int64    `json:"maxUploadBytes"`
	MaxPixels            *int64    `json:"maxPixels"`
	AllowedFormats       *[]string `json:"allowedFormats"`
	Dedupe               *bool     `json:"dedupe"`
	DefaultVisibility    *string   `json:"defaultVisibility"`
	TrashRetentionDays   *int      `json:"trashRetentionDays"`
	UploadsPerMinute     *int      `json:"uploadsPerMinute"`
	SessionIdleMinutes   *int      `json:"sessionIdleMinutes"`
	SessionAbsoluteHours *int      `json:"sessionAbsoluteHours"`
	PublicURL            *string   `json:"publicUrl"`
	ImageURL             *string   `json:"imageUrl"`
	HardMaxUploadBytes   *int64    `json:"hardMaxUploadBytes"`
}

const (
	minUploadBytes = 1 << 10
	minMaxPixels   = 1_000_000
	maxMaxPixels   = 200_000_000
)

func defaultSettings(hardMax int64) Settings {
	formats := make([]string, len(imaging.AllFormats))
	for i, f := range imaging.AllFormats {
		formats[i] = string(f)
	}
	up := int64(20 << 20)
	if up > hardMax {
		up = hardMax
	}
	return Settings{
		MaxUploadBytes:       up,
		MaxPixels:            50_000_000,
		AllowedFormats:       formats,
		Dedupe:               true,
		DefaultVisibility:    "public",
		TrashRetentionDays:   30,
		UploadsPerMinute:     60,
		SessionIdleMinutes:   30,
		SessionAbsoluteHours: 12,
	}
}

// validate checks every range from docs/api.md §3.
func (st *Settings) validate(hardMax int64) error {
	if st.MaxUploadBytes < minUploadBytes || st.MaxUploadBytes > hardMax {
		return validation("maxUploadBytes must be between %d and %d.", minUploadBytes, hardMax)
	}
	if st.MaxPixels < minMaxPixels || st.MaxPixels > maxMaxPixels {
		return validation("maxPixels must be between %d and %d.", minMaxPixels, maxMaxPixels)
	}
	if len(st.AllowedFormats) == 0 {
		return validation("allowedFormats must contain at least one format.")
	}
	seen := map[string]bool{}
	for _, f := range st.AllowedFormats {
		if _, ok := imaging.ParseFormat(f); !ok {
			return validation("Unknown format %q.", f)
		}
		if seen[f] {
			return validation("Duplicate format %q.", f)
		}
		seen[f] = true
	}
	if st.DefaultVisibility != "public" && st.DefaultVisibility != "private" {
		return validation("defaultVisibility must be public or private.")
	}
	if st.TrashRetentionDays < 0 || st.TrashRetentionDays > 365 {
		return validation("trashRetentionDays must be between 0 and 365.")
	}
	if st.UploadsPerMinute < 1 || st.UploadsPerMinute > 10000 {
		return validation("uploadsPerMinute must be between 1 and 10000.")
	}
	if st.SessionIdleMinutes < 5 || st.SessionIdleMinutes > 10080 {
		return validation("sessionIdleMinutes must be between 5 and 10080.")
	}
	if st.SessionAbsoluteHours < 1 || st.SessionAbsoluteHours > 8760 {
		return validation("sessionAbsoluteHours must be between 1 and 8760.")
	}
	return nil
}

func (st Settings) allows(f imaging.Format) bool {
	return slices.Contains(st.AllowedFormats, string(f))
}

// settingKeys maps persisted keys to accessors, keeping the JSON names in one place (the struct tags).
func (st Settings) toMap() map[string]any {
	b, _ := json.Marshal(st)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// loadSettings reads persisted values over the defaults. Invalid stored values fall back to the default
// for that key (and are logged), so a bad row can never lock the admin out.
func (s *Service) loadSettings(ctx context.Context) error {
	st := defaultSettings(s.cfg.HardMaxUpload)
	stored, err := s.db.AllSettings(ctx)
	if err != nil {
		return err
	}
	base := st.toMap()
	for k, raw := range stored {
		if _, known := base[k]; !known {
			continue
		}
		merged := st.toMap()
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			s.log.Warn("ignoring unreadable setting", "key", k)
			continue
		}
		merged[k] = v
		b, _ := json.Marshal(merged)
		var cand Settings
		if err := json.Unmarshal(b, &cand); err != nil || cand.validate(s.cfg.HardMaxUpload) != nil {
			s.log.Warn("ignoring invalid stored setting", "key", k)
			continue
		}
		st = cand
	}
	s.settings.Store(&st)
	s.applySettingSideEffects(st)
	return nil
}

func (s *Service) applySettingSideEffects(st Settings) {
	s.uploadLimit.Configure(float64(st.UploadsPerMinute)/60, st.UploadsPerMinute)
}

// Settings returns a copy of the current settings (no authorization: used internally and for limits).
func (s *Service) Settings() Settings {
	st := *s.settings.Load()
	st.AllowedFormats = slices.Clone(st.AllowedFormats)
	return st
}

// GetSettings is the admin view of the settings.
func (s *Service) GetSettings(p Principal) (SettingsView, error) {
	if !p.IsAdmin() {
		return SettingsView{}, s.denyAdmin(p)
	}
	return s.settingsView(), nil
}

func (s *Service) settingsView() SettingsView {
	return SettingsView{Settings: s.Settings(), PublicURL: s.cfg.PublicURL, ImageURL: s.cfg.ImageURL, HardMaxUploadBytes: s.cfg.HardMaxUpload}
}

// UpdateSettings validates and persists a partial update, then audits what changed.
func (s *Service) UpdateSettings(ctx context.Context, p Principal, patch SettingsPatch) (SettingsView, error) {
	if !p.IsAdmin() {
		return SettingsView{}, s.denyAdmin(p)
	}
	if patch.PublicURL != nil && *patch.PublicURL != s.cfg.PublicURL ||
		patch.ImageURL != nil && *patch.ImageURL != s.cfg.ImageURL ||
		patch.HardMaxUploadBytes != nil && *patch.HardMaxUploadBytes != s.cfg.HardMaxUpload {
		return SettingsView{}, validation("publicUrl, imageUrl and hardMaxUploadBytes are read-only (infrastructure configuration).")
	}
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	old := s.Settings()
	nw := old
	if patch.MaxUploadBytes != nil {
		nw.MaxUploadBytes = *patch.MaxUploadBytes
	}
	if patch.MaxPixels != nil {
		nw.MaxPixels = *patch.MaxPixels
	}
	if patch.AllowedFormats != nil {
		nw.AllowedFormats = slices.Clone(*patch.AllowedFormats)
	}
	if patch.Dedupe != nil {
		nw.Dedupe = *patch.Dedupe
	}
	if patch.DefaultVisibility != nil {
		nw.DefaultVisibility = *patch.DefaultVisibility
	}
	if patch.TrashRetentionDays != nil {
		nw.TrashRetentionDays = *patch.TrashRetentionDays
	}
	if patch.UploadsPerMinute != nil {
		nw.UploadsPerMinute = *patch.UploadsPerMinute
	}
	if patch.SessionIdleMinutes != nil {
		nw.SessionIdleMinutes = *patch.SessionIdleMinutes
	}
	if patch.SessionAbsoluteHours != nil {
		nw.SessionAbsoluteHours = *patch.SessionAbsoluteHours
	}
	if err := nw.validate(s.cfg.HardMaxUpload); err != nil {
		return SettingsView{}, err
	}

	oldMap, newMap := old.toMap(), nw.toMap()
	changed := map[string]string{}
	var summary []string
	for k, nv := range newMap {
		ob, _ := json.Marshal(oldMap[k])
		nb, _ := json.Marshal(nv)
		if string(ob) != string(nb) {
			changed[k] = string(nb)
			summary = append(summary, fmt.Sprintf("%s: %s -> %s", k, ob, nb))
		}
	}
	if len(changed) == 0 {
		return s.settingsView(), nil
	}
	if err := s.db.PutSettings(ctx, changed); err != nil {
		return SettingsView{}, err
	}
	s.settings.Store(&nw)
	s.applySettingSideEffects(nw)
	slices.Sort(summary)
	s.audit(ctx, p, "settings.update", "", strings.Join(summary, "; "))
	return s.settingsView(), nil
}
