package session

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type storeConnInfo struct {
	addr     string
	username string
	password string
	selectDB int
	useTLS   bool
}

func parseStoreURL(raw string) (storeConnInfo, error) {
	if strings.TrimSpace(raw) == "" {
		return storeConnInfo{}, errors.New("session store url is empty")
	}

	if !strings.Contains(raw, "://") {
		return parseStoreAddr(raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return storeConnInfo{}, fmt.Errorf("parse url: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return storeConnInfo{}, errors.New("session store host missing")
	}

	port := parsed.Port()
	if port == "" {
		port = "6379"
	}

	selectDB := 0
	if strings.TrimSpace(parsed.Path) != "" && parsed.Path != "/" {
		path := strings.TrimPrefix(parsed.Path, "/")
		db, err := strconv.Atoi(path)
		if err != nil {
			return storeConnInfo{}, fmt.Errorf("invalid session store db: %w", err)
		}
		if db < 0 {
			return storeConnInfo{}, errors.New("invalid session store db")
		}
		selectDB = db
	}

	username := ""
	password := ""
	if parsed.User != nil {
		username = parsed.User.Username()
		pw, _ := parsed.User.Password()
		password = pw
	}

	useTLS := strings.EqualFold(parsed.Scheme, "rediss")

	return storeConnInfo{
		addr:     net.JoinHostPort(host, port),
		username: username,
		password: password,
		selectDB: selectDB,
		useTLS:   useTLS,
	}, nil
}

func parseStoreAddr(addr string) (storeConnInfo, error) {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return storeConnInfo{}, errors.New("session store address is empty")
	}

	host, port, err := net.SplitHostPort(trimmed)
	if err != nil {
		var addrErr *net.AddrError
		if !errors.As(err, &addrErr) {
			return storeConnInfo{}, fmt.Errorf("invalid session store address: %w", err)
		}
		switch addrErr.Err {
		case "missing port in address":
			host = strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
			port = "6379"
		case "too many colons in address":
			host = trimmed
			port = "6379"
		default:
			return storeConnInfo{}, fmt.Errorf("invalid session store address: %w", err)
		}
	}

	if strings.TrimSpace(host) == "" {
		return storeConnInfo{}, errors.New("session store host missing")
	}

	return storeConnInfo{
		addr:     net.JoinHostPort(host, port),
		selectDB: 0,
		useTLS:   false,
	}, nil
}
