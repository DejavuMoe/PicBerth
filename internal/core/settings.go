package core

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"picberth/internal/imaging"
)

// Fixed limits that are no longer runtime settings (approved 2026-11-09).
const (
	maxPixels        = 50_000_000
	uploadsPerMinute = 60
)

// Settings are the runtime-editable business settings (docs/api.md §3).
type Settings struct {
	MaxUploadBytes     int64    `json:"maxUploadBytes"`
	AllowedFormats     []string `json:"allowedFormats"`
	Dedupe             bool     `json:"dedupe"`
	TrashRetentionDays int      `json:"trashRetentionDays"`
	AuditRetentionDays int      `json:"auditRetentionDays"`
}

// SettingsView is Settings plus the read-only infrastructure values.
type SettingsView struct {
	Settings
	PublicURL          string `json:"publicUrl"`
	ImageURL           string `json:"imageUrl"`
	HardMaxUploadBytes int64  `json:"hardMaxUploadBytes"`
}

// SettingsPatch is a partial update. Read-only fields are accepted only when they equal the current value
// (clients may echo a whole GET response back). Removed keys are unknown fields and are rejected.
type SettingsPatch struct {
	MaxUploadBytes     *int64    `json:"maxUploadBytes"`
	AllowedFormats     *[]string `json:"allowedFormats"`
	Dedupe             *bool     `json:"dedupe"`
	TrashRetentionDays *int      `json:"trashRetentionDays"`
	AuditRetentionDays *int      `json:"auditRetentionDays"`
	PublicURL          *string   `json:"publicUrl"`
	ImageURL           *string   `json:"imageUrl"`
	HardMaxUploadBytes *int64    `json:"hardMaxUploadBytes"`
}

const minUploadBytes = 1 << 10

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
		MaxUploadBytes:     up,
		AllowedFormats:     formats,
		Dedupe:             true,
		TrashRetentionDays: 30,
		AuditRetentionDays: 180,
	}
}

// validate checks every range from docs/api.md §3.
func (st *Settings) validate(hardMax int64) error {
	if st.MaxUploadBytes < minUploadBytes || st.MaxUploadBytes > hardMax {
		return validation("maxUploadBytes must be between %d and %d.", minUploadBytes, hardMax)
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
	if st.TrashRetentionDays < 0 || st.TrashRetentionDays > 365 {
		return validation("trashRetentionDays must be between 0 and 365.")
	}
	if st.AuditRetentionDays < 7 || st.AuditRetentionDays > 3650 {
		return validation("auditRetentionDays must be between 7 and 3650.")
	}
	return nil
}

func (st Settings) allows(f imaging.Format) bool {
	return slices.Contains(st.AllowedFormats, string(f))
}

// toMap exposes the JSON view so persisted values are matched by their JSON names.
func (st Settings) toMap() map[string]any {
	b, _ := json.Marshal(st)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

// loadSettings reads persisted values over the defaults. Invalid stored values fall back to the default
// for that key (and are logged), so a bad row can never lock the admin out. Keys that no longer exist as
// settings (from older schemas or hand edits) are ignored.
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
	return nil
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
	if patch.AllowedFormats != nil {
		nw.AllowedFormats = slices.Clone(*patch.AllowedFormats)
	}
	if patch.Dedupe != nil {
		nw.Dedupe = *patch.Dedupe
	}
	if patch.TrashRetentionDays != nil {
		nw.TrashRetentionDays = *patch.TrashRetentionDays
	}
	if patch.AuditRetentionDays != nil {
		nw.AuditRetentionDays = *patch.AuditRetentionDays
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
	slices.Sort(summary)
	s.audit(ctx, p, "settings.update", "", strings.Join(summary, "; "))
	return s.settingsView(), nil
}
