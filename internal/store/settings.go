package store

// Settings is a tiny key/value store for app state (wizard completion,
// preferences). Keys are app-scoped constants; values are short strings.
func (s *Store) SettingGet(key string) string {
	var v string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v); err != nil {
		return ""
	}
	return v
}

// SettingSet upserts a setting.
func (s *Store) SettingSet(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}
