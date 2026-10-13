package migrations

import (
	"encoding/json"
	"testing"
)

func TestNormalizeActivityTime(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"go raw with monotonic", "2026-10-12 14:45:55.709614141 +0500 +05 m=+9883.256736244", "2026-10-12T09:45:55Z"},
		{"go raw without fraction", "2026-10-12 14:45:55 +0500 +05", "2026-10-12T09:45:55Z"},
		{"go raw utc", "2026-09-02 12:00:00 +0000 UTC", "2026-09-02T12:00:00Z"},
		{"go raw negative offset", "2026-10-12 14:45:55.123 -0700 PST", "2026-10-12T21:45:55Z"},
		{"empty", "", "none"},
		{"none", "none", "none"},
		{"typed nil", "<nil>", "none"},
		{"nil", "nil", "none"},
		{"already rfc3339 z", "2026-10-12T09:45:55Z", "2026-10-12T09:45:55Z"},
		{"already rfc3339 offset", "2026-10-12T14:45:55+05:00", "2026-10-12T14:45:55+05:00"},
		{"unknown", "не дата", "не дата"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeActivityTime(tc.in); got != tc.want {
				t.Fatalf("normalizeActivityTime(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestActivityPersonLabel(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	cases := []struct {
		name     string
		first    string
		last     string
		username string
		want     string
	}{
		{"full", "Иван", "Иванов", "iivanov", "Иванов Иван (iivanov)"},
		{"no username", "Иван", "Иванов", "", "Иванов Иван"},
		{"no name", "", "", "iivanov", "iivanov"},
		{"username equals name", "Иванов", "", "Иванов", "Иванов"},
		{"empty", "", "", "", id},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := activityPersonLabel(tc.first, tc.last, tc.username, id); got != tc.want {
				t.Fatalf("activityPersonLabel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHumanizeActivityValues(t *testing.T) {
	const (
		userID    = "11111111-1111-1111-1111-111111111111"
		groupID   = "22222222-2222-2222-2222-222222222222"
		unknownID = "33333333-3333-3333-3333-333333333333"
		knownName = "Иванов Иван (iivanov)"
		groupName = "Первая линия"
	)
	h := &activityHumanizer{
		users:  map[string]string{userID: knownName},
		groups: map[string]string{groupID: groupName},
	}

	cases := []struct {
		name        string
		in          string
		want        string
		wantChanged bool
	}{
		{
			name:        "user id to label",
			in:          `{"assigned":"` + userID + `"}`,
			want:        `{"assigned":"` + knownName + `"}`,
			wantChanged: true,
		},
		{
			name:        "group id to name",
			in:          `{"group_changed":"` + groupID + `"}`,
			want:        `{"group_changed":"` + groupName + `"}`,
			wantChanged: true,
		},
		{
			name:        "time conversion",
			in:          `{"closed":"2026-10-12 14:45:55.5 +0500 +05 m=+1.2"}`,
			want:        `{"closed":"2026-10-12T09:45:55Z"}`,
			wantChanged: true,
		},
		{
			name:        "unknown uuid stays",
			in:          `{"assigned":"` + unknownID + `"}`,
			want:        `{"assigned":"` + unknownID + `"}`,
			wantChanged: false,
		},
		{
			name:        "already human",
			in:          `{"assigned":"` + knownName + `"}`,
			want:        `{"assigned":"` + knownName + `"}`,
			wantChanged: false,
		},
		{
			name:        "none and unknown keys untouched",
			in:          `{"assigned":"none","status":"in_progress"}`,
			want:        `{"assigned":"none","status":"in_progress"}`,
			wantChanged: false,
		},
		{
			name:        "non-string value untouched",
			in:          `{"assigned":123}`,
			want:        `{"assigned":123}`,
			wantChanged: false,
		},
		{
			name:        "mixed",
			in:          `{"assigned":"` + userID + `","status":"in_progress"}`,
			want:        `{"assigned":"` + knownName + `","status":"in_progress"}`,
			wantChanged: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed, err := humanizeActivityValues([]byte(tc.in), h)
			if err != nil {
				t.Fatalf("humanizeActivityValues() error: %v", err)
			}
			if changed != tc.wantChanged {
				t.Fatalf("changed = %v, want %v", changed, tc.wantChanged)
			}
			if !jsonEqual(t, got, tc.want) {
				t.Fatalf("humanizeActivityValues() = %s, want %s", got, tc.want)
			}
		})
	}

	if _, changed, err := humanizeActivityValues(nil, h); err != nil || changed {
		t.Fatalf("nil input: changed=%v err=%v, want false/nil", changed, err)
	}
	if _, changed, err := humanizeActivityValues([]byte("null"), h); err != nil || changed {
		t.Fatalf("null input: changed=%v err=%v, want false/nil", changed, err)
	}
}

func TestResolveActivityLabel(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	labels := map[string]string{id: "Иванов Иван (iivanov)"}

	if got := resolveActivityLabel(labels, id); got != "Иванов Иван (iivanov)" {
		t.Fatalf("known id = %q", got)
	}
	if got := resolveActivityLabel(labels, "none"); got != "none" {
		t.Fatalf("none = %q", got)
	}
	if got := resolveActivityLabel(labels, "не-id"); got != "не-id" {
		t.Fatalf("non-id = %q", got)
	}
	other := "44444444-4444-4444-4444-444444444444"
	if got := resolveActivityLabel(labels, other); got != other {
		t.Fatalf("unknown id = %q, want raw", got)
	}
}

func jsonEqual(t *testing.T, a []byte, b string) bool {
	t.Helper()
	var av, bv any
	if err := json.Unmarshal(a, &av); err != nil {
		t.Fatalf("failed to unmarshal actual %s: %v", a, err)
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		t.Fatalf("failed to unmarshal expected %s: %v", b, err)
	}
	ab, _ := json.Marshal(av)
	bb, _ := json.Marshal(bv)
	return string(ab) == string(bb)
}
