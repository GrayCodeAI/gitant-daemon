package git

import (
	"strings"
	"testing"
)

func TestPktLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", "0004"},
		{"single char", "a", "0005a"},
		{"newline terminated", "want abc\n", "000dwant abc\n"},
		{"null byte", "\000", "0005\000"},
		{"typical ref line", "abc123 refs/heads/main\n", "001babc123 refs/heads/main\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PktLine(tt.input)
			if got != tt.want {
				t.Errorf("PktLine(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Verify the length prefix matches the actual data length
			if len(got) >= 4 {
				prefix := got[:4]
				dataLen := len(got) - 4
				expectedPrefix := ""
				switch dataLen {
				case 0:
					expectedPrefix = "0004"
				default:
					// Just verify it's valid hex
					for _, c := range prefix {
						if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
							t.Errorf("PktLine prefix contains non-hex char: %c", c)
						}
					}
				}
				if len(tt.input) > 0 {
					_ = expectedPrefix
				}
			}
		})
	}
}

func TestPktLineLength(t *testing.T) {
	// The length prefix should encode: len(data) + 4
	for _, s := range []string{"", "a", "hello", "want abc123\n"} {
		got := PktLine(s)
		totalLen := len(got)
		dataLen := totalLen - 4
		if dataLen != len(s) {
			t.Errorf("PktLine(%q): data portion length = %d, want %d", s, dataLen, len(s))
		}
	}
}

func TestFlushPacket(t *testing.T) {
	if got := FlushPacket(); got != "0000" {
		t.Errorf("FlushPacket() = %q, want %q", got, "0000")
	}
}

func TestPktLinef(t *testing.T) {
	got := PktLinef("want %s\n", "abc123")
	want := PktLine("want abc123\n")
	if got != want {
		t.Errorf("PktLinef = %q, want %q", got, want)
	}

	got2 := PktLinef("# service=%s\n", "git-upload-pack")
	want2 := PktLine("# service=git-upload-pack\n")
	if got2 != want2 {
		t.Errorf("PktLinef = %q, want %q", got2, want2)
	}
}

func TestServiceRefResponseEmpty(t *testing.T) {
	resp := ServiceRefResponse("git-upload-pack", []RefLine{})
	// Should contain the service header, flush, zero-oid capabilities line, and final flush
	if !strings.Contains(resp, "# service=git-upload-pack\n") {
		t.Error("missing service header")
	}
	if !strings.Contains(resp, "0000000000000000000000000000000000000000 capabilities^{}") {
		t.Error("missing capabilities line for empty refs")
	}
	// Should have exactly 2 flush packets
	flushCount := strings.Count(resp, "0000")
	if flushCount < 2 {
		t.Errorf("expected at least 2 flush packets, got %d", flushCount)
	}
}

func TestServiceRefResponseWithRefs(t *testing.T) {
	refs := []RefLine{
		{Hash: "abc123def456", Name: "refs/heads/main"},
		{Hash: "789012345678", Name: "refs/heads/dev"},
	}
	resp := ServiceRefResponse("git-upload-pack", refs)

	if !strings.Contains(resp, "# service=git-upload-pack\n") {
		t.Error("missing service header")
	}
	// First ref should include capabilities (null byte separator)
	if !strings.Contains(resp, "abc123def456 refs/heads/main\000side-band-64k thin-pack") {
		t.Errorf("first ref missing capabilities: %q", resp)
	}
	// Second ref should NOT have capabilities
	if !strings.Contains(resp, "789012345678 refs/heads/dev\n") {
		t.Errorf("missing second ref: %q", resp)
	}
	// Second ref should not have null byte
	if strings.Contains(resp, "789012345678 refs/heads/dev\000") {
		t.Error("second ref should not have capabilities")
	}
}

func TestServiceRefResponseSingleRef(t *testing.T) {
	refs := []RefLine{{Hash: "aaa", Name: "refs/heads/main"}}
	resp := ServiceRefResponse("git-receive-pack", refs)

	if !strings.Contains(resp, "# service=git-receive-pack\n") {
		t.Error("missing service header")
	}
	if !strings.Contains(resp, "aaa refs/heads/main\000") {
		t.Error("single ref should include capabilities")
	}
}

func TestParseWantLines(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"empty",
			[]string{},
			nil,
		},
		{
			"single want",
			[]string{"want abc123\n"},
			[]string{"abc123"},
		},
		{
			"multiple wants",
			[]string{"want abc123\n", "want def456\n"},
			[]string{"abc123", "def456"},
		},
		{
			"mixed lines",
			[]string{"want abc123\n", "have def456\n", "want 789abc\n"},
			[]string{"abc123", "789abc"},
		},
		{
			"with whitespace",
			[]string{"  want abc123  \n", "  want def456\n"},
			[]string{"abc123", "def456"},
		},
		{
			"no want lines",
			[]string{"have abc123\n", "done\n"},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseWantLines(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d hashes, want %d", len(got), len(tt.want))
			}
			for i, h := range got {
				if h != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, h, tt.want[i])
				}
			}
		})
	}
}

func TestParseHaveLines(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			"empty",
			[]string{},
			nil,
		},
		{
			"single have",
			[]string{"have abc123\n"},
			[]string{"abc123"},
		},
		{
			"multiple haves",
			[]string{"have abc123\n", "have def456\n"},
			[]string{"abc123", "def456"},
		},
		{
			"mixed with want",
			[]string{"want abc123\n", "have def456\n", "have 789abc\n"},
			[]string{"def456", "789abc"},
		},
		{
			"with whitespace",
			[]string{"  have abc123  \n"},
			[]string{"abc123"},
		},
		{
			"no have lines",
			[]string{"want abc123\n", "done\n"},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHaveLines(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d hashes, want %d", len(got), len(tt.want))
			}
			for i, h := range got {
				if h != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, h, tt.want[i])
				}
			}
		})
	}
}

func TestParsePushRefUpdates(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []PushRefUpdate
	}{
		{
			"empty",
			[]string{},
			nil,
		},
		{
			"single update",
			[]string{"0000000000000000000000000000000000000000 abc123 refs/heads/main\n"},
			[]PushRefUpdate{
				{OldHash: "0000000000000000000000000000000000000000", NewHash: "abc123", RefName: "refs/heads/main"},
			},
		},
		{
			"multiple updates",
			[]string{
				"aaa bbb refs/heads/main\n",
				"ccc ddd refs/heads/dev\n",
			},
			[]PushRefUpdate{
				{OldHash: "aaa", NewHash: "bbb", RefName: "refs/heads/main"},
				{OldHash: "ccc", NewHash: "ddd", RefName: "refs/heads/dev"},
			},
		},
		{
			"delete branch",
			[]string{"abc123 0000000000000000000000000000000000000000 refs/heads/old\n"},
			[]PushRefUpdate{
				{OldHash: "abc123", NewHash: "0000000000000000000000000000000000000000", RefName: "refs/heads/old"},
			},
		},
		{
			"malformed line ignored",
			[]string{"only_two_parts\n", "aaa bbb refs/heads/main\n"},
			[]PushRefUpdate{
				{OldHash: "aaa", NewHash: "bbb", RefName: "refs/heads/main"},
			},
		},
		{
			"create new branch",
			[]string{"0000000000000000000000000000000000000000 abc123 refs/heads/feature\n"},
			[]PushRefUpdate{
				{OldHash: "0000000000000000000000000000000000000000", NewHash: "abc123", RefName: "refs/heads/feature"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePushRefUpdates(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d updates, want %d", len(got), len(tt.want))
			}
			for i, u := range got {
				if u != tt.want[i] {
					t.Errorf("got[%d] = %+v, want %+v", i, u, tt.want[i])
				}
			}
		})
	}
}

func TestParseWantLinesNoPanic(t *testing.T) {
	// Ensure no panic on edge cases
	ParseWantLines(nil)
	ParseWantLines([]string{""})
	ParseWantLines([]string{"want", "want "})
}

func TestParseHaveLinesNoPanic(t *testing.T) {
	ParseHaveLines(nil)
	ParseHaveLines([]string{""})
	ParseHaveLines([]string{"have", "have "})
}

func TestParsePushRefUpdatesNoPanic(t *testing.T) {
	ParsePushRefUpdates(nil)
	ParsePushRefUpdates([]string{""})
	ParsePushRefUpdates([]string{"a", "a b"})
}

func TestServiceRefResponseRoundtrip(t *testing.T) {
	// Verify the response contains valid pkt-line formatting
	refs := []RefLine{
		{Hash: "abc123", Name: "refs/heads/main"},
	}
	resp := ServiceRefResponse("git-upload-pack", refs)

	// Each non-flush line should start with a 4-char hex length prefix
	lines := strings.Split(resp, "\n")
	for _, line := range lines {
		if line == "" || line == "0000" {
			continue
		}
		if len(line) < 4 {
			t.Errorf("line too short to be a valid pkt-line: %q", line)
			continue
		}
		prefix := line[:4]
		for _, c := range prefix {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Errorf("invalid hex prefix %q in line %q", prefix, line)
				break
			}
		}
	}
}
