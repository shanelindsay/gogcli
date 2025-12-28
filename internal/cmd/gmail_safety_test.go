package cmd

import "testing"

func TestGmailAllowlistAllows(t *testing.T) {
	allowlist := parseAllowlistEntries([]string{
		"alice@example.com",
		"@example.org",
		"*.lab.example.net",
	})

	cases := []struct {
		email string
		want  bool
	}{
		{"alice@example.com", true},
		{"Alice <alice@example.com>", true},
		{"alice+test@example.com", true},
		{"bob@example.org", true},
		{"carol@sub.lab.example.net", true},
		{"evil@evil.com", false},
	}
	for _, tc := range cases {
		if got := allowlist.allows(tc.email); got != tc.want {
			t.Fatalf("allows(%q)=%v want %v", tc.email, got, tc.want)
		}
	}
}

func TestGmailAllowlistBlockedRecipients(t *testing.T) {
	allowlist := parseAllowlistEntries([]string{"@example.org"})
	blocked := blockedRecipients(allowlist, []string{"a@example.org", "b@evil.com"})
	if len(blocked) != 1 || blocked[0] != "b@evil.com" {
		t.Fatalf("unexpected blocked=%v", blocked)
	}
}
