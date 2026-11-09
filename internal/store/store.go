package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"net-merge/internal/rules"

	_ "modernc.org/sqlite"
)

// Settings is the desktop state that is not a routing rule.
type Settings struct {
	AdapterDown    map[string]float64 `json:"adapterDown,omitempty"`
	AdapterUp      map[string]float64 `json:"adapterUp,omitempty"`
	Theme          string             `json:"theme"` // dark or light
	StartWindows   bool               `json:"startWindows"`
	StartRouting   bool               `json:"startRouting"`
	AutoUpdate     bool               `json:"autoUpdate"`
	DefaultAdapter string             `json:"defaultAdapter,omitempty"`
	DownMbps       float64            `json:"downMbps,omitempty"`
	UpMbps         float64            `json:"upMbps,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{Theme: "dark", StartRouting: true}
}

// Store is the versioned config and usage history.
type Store struct {
	db *sql.DB
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "multiroute.db"))
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// OpenReadOnly reads saved configuration without creating or migrating a store.
func OpenReadOnly(dir string) (*Store, error) {
	path, err := filepath.Abs(filepath.Join(dir, "multiroute.db"))
	if err != nil {
		return nil, err
	}
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS meta (k TEXT PRIMARY KEY, v TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS samples (
  ts INTEGER NOT NULL,
  exe TEXT NOT NULL,
  path TEXT NOT NULL,
  adapter TEXT NOT NULL,
  down INTEGER NOT NULL,
  up INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS samples_ts ON samples(ts);
CREATE TABLE IF NOT EXISTS adapter_usage (
  name TEXT NOT NULL,
  period TEXT NOT NULL,
  down INTEGER NOT NULL,
  up INTEGER NOT NULL,
  PRIMARY KEY(name, period)
);
`)
	if err != nil {
		return err
	}
	var v string
	err = s.db.QueryRow(`SELECT v FROM meta WHERE k='schema'`).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.db.Exec(`INSERT INTO meta(k,v) VALUES('schema','1')`)
	}
	return err
}

func (s *Store) Rules() ([]rules.Rule, error) {
	var raw string
	err := s.db.QueryRow(`SELECT v FROM meta WHERE k='rules'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var list []rules.Rule
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Store) SaveRules(list []rules.Rule) error {
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO meta(k,v) VALUES('rules',?) ON CONFLICT(k) DO UPDATE SET v=excluded.v`, string(b))
	return err
}

func (s *Store) LoadSettings() (Settings, error) {
	st := DefaultSettings()
	var raw string
	err := s.db.QueryRow(`SELECT v FROM meta WHERE k='settings'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal([]byte(raw), &st); err != nil {
		return DefaultSettings(), err
	}
	if st.Theme == "" {
		st.Theme = "dark"
	}
	return st, nil
}

func (s *Store) SaveSettings(st Settings) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO meta(k,v) VALUES('settings',?) ON CONFLICT(k) DO UPDATE SET v=excluded.v`, string(b))
	return err
}

// AddSample records bytes the engine routed for one app and adapter.
func (s *Store) AddSample(ts time.Time, exe, path, adapter string, down, up int64) error {
	if exe == "" {
		exe = "Unknown"
	}
	_, err := s.db.Exec(`INSERT INTO samples(ts,exe,path,adapter,down,up) VALUES(?,?,?,?,?,?)`,
		ts.Unix(), exe, path, adapter, down, up)
	return err
}

// UsageRow is one app total inside a time range.
type UsageRow struct {
	Exe       string           `json:"exe"`
	Path      string           `json:"path"`
	Down      int64            `json:"down"`
	Up        int64            `json:"up"`
	ByNIC     map[string]int64 `json:"byNic"`
	ByNICDown map[string]int64 `json:"byNicDown"`
	ByNICUp   map[string]int64 `json:"byNicUp"`
}

func (s *Store) Usage(from time.Time) ([]UsageRow, error) {
	rows, err := s.db.Query(`SELECT exe, path, adapter, down, up FROM samples WHERE ts>=?`, from.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type acc struct {
		UsageRow
	}
	order := []string{}
	by := map[string]*acc{}
	for rows.Next() {
		var exe, path, adapter string
		var down, up int64
		if err := rows.Scan(&exe, &path, &adapter, &down, &up); err != nil {
			return nil, err
		}
		a := by[exe]
		if a == nil {
			a = &acc{UsageRow: UsageRow{Exe: exe, Path: path, ByNIC: map[string]int64{}, ByNICDown: map[string]int64{}, ByNICUp: map[string]int64{}}}
			by[exe] = a
			order = append(order, exe)
		}
		if path != "" {
			a.Path = path
		}
		a.Down += down
		a.Up += up
		a.ByNIC[adapter] += down + up
		a.ByNICDown[adapter] += down
		a.ByNICUp[adapter] += up
	}
	out := make([]UsageRow, 0, len(order))
	for _, exe := range order {
		out = append(out, by[exe].UsageRow)
	}
	return out, rows.Err()
}

// PeriodBytes is the download plus upload counted against a cap key.
func (s *Store) AdapterBytes(name, period string) (int64, int64, error) {
	var down, up sql.NullInt64
	err := s.db.QueryRow(`SELECT down, up FROM adapter_usage WHERE name=? AND period=?`, name, period).Scan(&down, &up)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return down.Int64, up.Int64, err
}

func (s *Store) SaveAdapterBytes(name, period string, down, up int64) error {
	_, err := s.db.Exec(`INSERT INTO adapter_usage(name,period,down,up) VALUES(?,?,?,?)
		ON CONFLICT(name,period) DO UPDATE SET down=excluded.down, up=excluded.up`, name, period, down, up)
	return err
}

func (s *Store) PeriodBytes(exe string, from time.Time) (int64, error) {
	var n sql.NullInt64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(down+up),0) FROM samples WHERE lower(exe)=lower(?) AND ts>=?`, exe, from.Unix()).Scan(&n)
	return n.Int64, err
}
