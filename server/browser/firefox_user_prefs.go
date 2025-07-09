package browser

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cast"

	"github.com/starudream/aichat-proxy/server/config"
	"github.com/starudream/aichat-proxy/server/internal/json"
	"github.com/starudream/aichat-proxy/server/logger"
)

type UserPrefs struct {
	Raw  map[string]string
	Flat map[string]any
}

var reUserPref = regexp.MustCompile(`^user_pref\("([^"]+)",\s*(.*)\);$`)

func GetUserPrefs() *UserPrefs {
	bs, err := os.ReadFile(filepath.Join(config.Userdata0Path, "prefs.js"))
	if err != nil {
		logger.Error().Err(err).Msg("read user prefs.js error")
		return nil
	}
	up := &UserPrefs{Raw: map[string]string{}, Flat: map[string]any{}}
	for sc := bufio.NewScanner(bytes.NewReader(bs)); sc.Scan(); {
		ss := reUserPref.FindStringSubmatch(sc.Text())
		if len(ss) != 3 {
			continue
		}
		k, v := ss[1], ss[2]
		up.Raw[k] = v
		if strings.HasPrefix(v, `"`) {
			v = v[1 : len(v)-1]
			if v != "" && v[0] == '[' {
				s, e1 := json.UnmarshalTo[string](`"` + v + `"`)
				if e1 == nil {
					t, e2 := json.UnmarshalTo[[]any](s)
					if e2 == nil {
						up.Flat[k] = t
					}
				}
			} else if v != "" && v[0] == '{' {
				s, e1 := json.UnmarshalTo[string](`"` + v + `"`)
				if e1 == nil {
					t, e2 := json.UnmarshalTo[map[string]any](s)
					if e2 == nil {
						up.Flat[k] = t
					}
				}
			}
			if _, ok := up.Flat[k]; !ok {
				up.Flat[k] = v
			}
		} else if v == "true" || v == "false" {
			up.Flat[k] = v == "true"
		} else {
			up.Flat[k] = cast.To[float64](v)
		}
	}
	return up
}

func (up *UserPrefs) GetExtId(name string) string {
	if v, ok1 := up.Flat["extensions.webextensions.uuids"]; ok1 {
		if m, ok2 := v.(map[string]any); ok2 {
			if id, ok3 := m[name]; ok3 {
				return id.(string)
			}
		}
	}
	return ""
}
