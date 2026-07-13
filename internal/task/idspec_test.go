package task

import (
	"reflect"
	"testing"
)

func TestParseIDSpec(t *testing.T) {
	cases := []struct {
		in      []string
		want    []int64
		wantErr bool
	}{
		{[]string{"5"}, []int64{5}, false},
		{[]string{"1", "2", "3"}, []int64{1, 2, 3}, false},
		{[]string{"1,5"}, []int64{1, 5}, false},
		{[]string{"1-3"}, []int64{1, 2, 3}, false},
		{[]string{"1-3,5"}, []int64{1, 2, 3, 5}, false},
		{[]string{"3-1"}, []int64{1, 2, 3}, false},      // reversed range normalizes
		{[]string{"2", "1-3"}, []int64{2, 1, 3}, false}, // dedup, first-seen order
		{[]string{}, nil, true},
		{[]string{"x"}, nil, true},
		{[]string{"1-x"}, nil, true},
		{[]string{"1-99999"}, nil, true}, // range too large
	}
	for _, c := range cases {
		got, err := ParseIDSpec(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseIDSpec(%v): expected error, got %v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseIDSpec(%v): %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("ParseIDSpec(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseTokensNegationAndStatus(t *testing.T) {
	_, f, text, err := ParseTokens([]string{"-urgente", "+casa", "status:done", "resto"}, base)
	if err != nil {
		t.Fatalf("ParseTokens: %v", err)
	}
	if len(f.ExcludeTags) != 1 || f.ExcludeTags[0] != "urgente" {
		t.Fatalf("expected ExcludeTags=[urgente], got %v", f.ExcludeTags)
	}
	if len(f.Tags) != 1 || f.Tags[0] != "casa" {
		t.Fatalf("expected Tags=[casa], got %v", f.Tags)
	}
	if f.Status != StatusDone {
		t.Fatalf("expected Status=done, got %q", f.Status)
	}
	if text != "resto" {
		t.Fatalf("expected free text 'resto', got %q", text)
	}

	// -5 and a lone - are free text, not exclusions
	_, f2, text2, _ := ParseTokens([]string{"-5", "-"}, base)
	if len(f2.ExcludeTags) != 0 {
		t.Fatalf("'-5'/'-' should not be exclude tags, got %v", f2.ExcludeTags)
	}
	if text2 != "-5 -" {
		t.Fatalf("expected '-5 -' as text, got %q", text2)
	}

	// -tag must never become a task attribute on add
	tk, _, _, _ := ParseTokens([]string{"comprar", "-urgente"}, base)
	if len(tk.Tags) != 0 {
		t.Fatalf("exclusion must not add a task tag, got %v", tk.Tags)
	}

	if _, _, _, err := ParseTokens([]string{"status:bogus"}, base); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestMergeTags(t *testing.T) {
	cases := []struct {
		existing, add, remove, want []string
	}{
		{[]string{"casa"}, []string{"urgente"}, nil, []string{"casa", "urgente"}},   // add keeps old
		{[]string{"casa", "urgente"}, nil, []string{"urgente"}, []string{"casa"}},   // remove
		{[]string{"casa"}, []string{"casa", "nova"}, nil, []string{"casa", "nova"}}, // dedup
		{[]string{"a", "b"}, []string{"c"}, []string{"b"}, []string{"a", "c"}},      // add + remove
		{nil, nil, []string{"x"}, nil},                                              // nothing
	}
	for i, c := range cases {
		got := MergeTags(c.existing, c.add, c.remove)
		if len(got) != len(c.want) {
			t.Errorf("case %d: got %v, want %v", i, got, c.want)
			continue
		}
		for j := range got {
			if got[j] != c.want[j] {
				t.Errorf("case %d: got %v, want %v", i, got, c.want)
				break
			}
		}
	}
}
