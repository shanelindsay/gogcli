package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"

	"github.com/steipete/gogcli/internal/config"
	"github.com/steipete/gogcli/internal/ui"
)

type allowlistMode int

const (
	allowlistEnforce allowlistMode = iota
	allowlistWarn
	allowlistOff
)

type gmailAllowlist struct {
	emails        map[string]struct{}
	domains       map[string]struct{}
	suffixDomains []string
}

func (a *gmailAllowlist) empty() bool {
	if a == nil {
		return true
	}
	return len(a.emails) == 0 && len(a.domains) == 0 && len(a.suffixDomains) == 0
}

func (a *gmailAllowlist) allows(email string) bool {
	email = normalizeEmailAddress(email)
	if email == "" || a == nil {
		return false
	}
	if _, ok := a.emails[email]; ok {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at == -1 || at+1 >= len(email) {
		return false
	}
	local := email[:at]
	domain := email[at+1:]
	if plus := strings.Index(local, "+"); plus != -1 {
		base := strings.TrimSpace(local[:plus])
		if base != "" {
			if _, ok := a.emails[base+"@"+domain]; ok {
				return true
			}
		}
	}
	if _, ok := a.domains[domain]; ok {
		return true
	}
	for _, suffix := range a.suffixDomains {
		if domain == suffix || strings.HasSuffix(domain, "."+suffix) {
			return true
		}
	}
	return false
}

func gmailAllowlistMode() (allowlistMode, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("GOG_GMAIL_ALLOWLIST_MODE")))
	switch mode {
	case "", "enforce":
		return allowlistEnforce, nil
	case "warn":
		return allowlistWarn, nil
	case "off":
		return allowlistOff, nil
	default:
		return allowlistOff, fmt.Errorf("invalid GOG_GMAIL_ALLOWLIST_MODE: %q (use enforce|warn|off)", mode)
	}
}

func loadGmailAllowlist() (*gmailAllowlist, string, error) {
	entries := make([]string, 0)
	var sources []string

	if raw := strings.TrimSpace(os.Getenv("GOG_GMAIL_ALLOWLIST")); raw != "" {
		entries = append(entries, splitCSV(raw)...)
		sources = append(sources, "GOG_GMAIL_ALLOWLIST")
	}

	path := strings.TrimSpace(os.Getenv("GOG_GMAIL_ALLOWLIST_FILE"))
	if path == "" {
		defaultPath, err := config.GmailAllowlistPath()
		if err != nil {
			return nil, "", err
		}
		path = defaultPath
	}
	if path != "" {
		if fileEntries, err := readAllowlistFile(path); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, "", err
			}
		} else if len(fileEntries) > 0 {
			entries = append(entries, fileEntries...)
			sources = append(sources, path)
		}
	}

	return parseAllowlistEntries(entries), strings.Join(sources, ", "), nil
}

func parseAllowlistEntries(entries []string) *gmailAllowlist {
	out := &gmailAllowlist{
		emails:        make(map[string]struct{}),
		domains:       make(map[string]struct{}),
		suffixDomains: make([]string, 0),
	}
	for _, entry := range entries {
		normalized := strings.ToLower(strings.TrimSpace(entry))
		if normalized == "" {
			continue
		}
		if strings.HasPrefix(normalized, "mailto:") {
			normalized = strings.TrimPrefix(normalized, "mailto:")
		}
		switch {
		case strings.HasPrefix(normalized, "*.") || strings.HasPrefix(normalized, "."):
			normalized = strings.TrimPrefix(strings.TrimPrefix(normalized, "*."), ".")
			if normalized != "" {
				out.suffixDomains = append(out.suffixDomains, normalized)
			}
		case strings.HasPrefix(normalized, "@"):
			normalized = strings.TrimPrefix(normalized, "@")
			if normalized != "" {
				out.domains[normalized] = struct{}{}
			}
		case strings.Contains(normalized, "@"):
			out.emails[normalized] = struct{}{}
		default:
			out.domains[normalized] = struct{}{}
		}
	}
	return out
}

func readAllowlistFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			entries = append(entries, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func normalizeEmailAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if addr, err := mail.ParseAddress(value); err == nil && addr != nil && addr.Address != "" {
		value = addr.Address
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func extractEmails(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if addrs, err := mail.ParseAddressList(v); err == nil && len(addrs) > 0 {
			for _, addr := range addrs {
				email := normalizeEmailAddress(addr.Address)
				if email != "" {
					out = append(out, email)
				}
			}
			continue
		}
		if email := normalizeEmailAddress(v); email != "" {
			out = append(out, email)
		}
	}
	return out
}

func blockedRecipients(allowlist *gmailAllowlist, recipients []string) []string {
	recipients = extractEmails(recipients)
	if len(recipients) == 0 {
		return nil
	}
	blocked := make([]string, 0)
	for _, r := range recipients {
		if !allowlist.allows(r) {
			blocked = append(blocked, r)
		}
	}
	return blocked
}

func checkGmailAllowlist(u *ui.UI, recipients []string) error {
	mode, err := gmailAllowlistMode()
	if err != nil {
		return err
	}
	if mode == allowlistOff {
		return nil
	}
	allowlist, source, err := loadGmailAllowlist()
	if err != nil {
		return err
	}
	if allowlist.empty() {
		if source == "" {
			source = "GOG_GMAIL_ALLOWLIST or ~/.config/gogcli/gmail-allowlist.txt"
		}
		return fmt.Errorf("gmail allowlist empty; configure %s", source)
	}
	blocked := blockedRecipients(allowlist, recipients)
	if len(blocked) == 0 {
		return nil
	}
	msg := fmt.Sprintf("blocked recipient(s) not in allowlist: %s", strings.Join(blocked, ", "))
	if mode == allowlistWarn {
		if u != nil {
			u.Err().Printf("WARN: %s", msg)
		}
		return nil
	}
	return errors.New(msg)
}

func requireGmailSendArm() error {
	if strings.TrimSpace(os.Getenv("GOG_GMAIL_REQUIRE_ARM")) != "1" {
		return nil
	}
	if strings.TrimSpace(os.Getenv("GOG_GMAIL_SEND_ARMED")) == "1" {
		return nil
	}
	return errors.New("gmail send guard not armed (set GOG_GMAIL_SEND_ARMED=1 for this process)")
}
