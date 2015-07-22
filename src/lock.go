package imoslock

import (
	"appengine"
	"appengine/datastore"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type LockConfig struct {
	Key              string
	Owner            string
	DurationInMillis int64
	Unlock           int64
	LockTime         int64
}

type Lock struct {
	Owner        string `json:"owner"`
	LockTime     int64  `json:"lock_time"`
	ModifiedTime int64  `json:"modified_time"`
}

type LockResult struct {
	Acquired bool  `json:"acquired"`
	Lock     *Lock `json:"lock"`
}

func init() {
	http.HandleFunc("/lock", lockHandler)
}

func GetKey(c appengine.Context, key string) *datastore.Key {
	return datastore.NewKey(c, "Lock", key, 0, nil)
}

func tryLock(c appengine.Context, cfg *LockConfig) (*LockResult, error) {
	result := &LockResult{}
	var l Lock
	if err := datastore.Get(c, GetKey(c, cfg.Key), &l); err != nil && err != datastore.ErrNoSuchEntity {
		return result, err
	}
	result.Lock = &l
	if time.Now().UnixNano() < l.LockTime {
		return result, nil
	}
	if cfg.Unlock != 0 && cfg.Unlock != l.LockTime {
		return result, nil
	}
	if cfg.DurationInMillis <= 0 {
		l.LockTime = 0
	} else {
		l.LockTime = time.Now().UnixNano() + cfg.DurationInMillis*1000000
	}
	l.Owner = cfg.Owner
	l.ModifiedTime = time.Now().UnixNano()
	if _, err := datastore.Put(c, GetKey(c, cfg.Key), &l); err != nil {
		return result, err
	}
	result.Acquired = true
	return result, nil
}

func lock(w http.ResponseWriter, r *http.Request) error {
	cfg := &LockConfig{}

	if r.FormValue("key") == "" {
		return errors.New("key is missing.")
	}
	cfg.Key = r.FormValue("key")

	if r.FormValue("owner") == "" {
		return errors.New("owner is missing.")
	}
	cfg.Owner = r.FormValue("owner")

	if r.FormValue("unlock") == "" && r.FormValue("duration") == "" {
		return errors.New("duration is missing.")
	}
	duration_in_secs, err := strconv.ParseFloat(r.FormValue("duration"), 32)
	if err != nil {
		return fmt.Errorf("Failed to convert duration: %s", err)
	}

	if r.FormValue("unlock") == "" {
		cfg.DurationInMillis = int64(duration_in_secs * 1000)
	} else {
		var err error
		cfg.Unlock, err = strconv.ParseInt(r.FormValue("unlock"), 10, 64)
		if err != nil {
			return fmt.Errorf("Failed to convert unlock: %s", err)
		}
		if cfg.Unlock == 0 {
			return errors.New("unlock must not be 0.")
		}
	}

	c := appengine.NewContext(r)
	var result *LockResult
	if err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		result, err = tryLock(c, cfg)
		return nil
	}, nil); err != nil {
		return fmt.Errorf("Failed to lock: %s", err)
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("Failed to build a json: %s", err)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write(bytes)
	return nil
}

func lockHandler(w http.ResponseWriter, r *http.Request) {
	err := lock(w, r)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err)
		return
	}
}
