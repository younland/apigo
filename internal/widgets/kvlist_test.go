package widgets

import "testing"

func TestFlushStartsWithOneEmptyRow(t *testing.T) {
	var s State
	s.Flush()
	if len(s.Rows) != 1 || s.Rows[0].filled() {
		t.Fatalf("rows %+v, want one empty row to type into", s.Rows)
	}
}

func TestFlushAppendsAfterAFilledRow(t *testing.T) {
	s := State{Rows: []Row{{Enabled: true, Key: "q", Value: "go"}}}
	s.Flush()
	if len(s.Rows) != 2 || s.Rows[1].filled() {
		t.Fatalf("rows %+v, want a new empty row after the filled one", s.Rows)
	}
}

func TestFlushKeepsOneEmptyRow(t *testing.T) {
	s := State{Rows: []Row{{Enabled: true, Key: "q"}, {Enabled: true}, {Enabled: true}}}
	s.Flush()
	if len(s.Rows) != 2 {
		t.Fatalf("rows %+v, want the extra empty rows dropped", s.Rows)
	}
}

func TestAddAndRemove(t *testing.T) {
	s := State{Rows: []Row{{Enabled: true, Key: "a", Value: "1"}, {Enabled: true}}}
	s.add = true
	s.Flush()
	if len(s.Rows) != 2 || s.Rows[0].Key != "a" || s.Rows[1].filled() {
		t.Fatalf("rows %+v after add, want the filled row and one empty row", s.Rows)
	}

	s.Rows = append(s.Rows, Row{Enabled: true, Key: "b", Value: "2"})
	s.remove = 1
	s.Flush()
	for _, r := range s.Rows {
		if r.Key == "a" {
			t.Fatalf("rows %+v after removing the first, want a gone", s.Rows)
		}
	}
	if len(s.Rows) != 3 || s.Rows[1].Key != "b" {
		t.Fatalf("rows %+v after removing the first, want b after the empty row", s.Rows)
	}
}

func TestEnabledSkipsDisabledAndBlank(t *testing.T) {
	s := State{Rows: []Row{
		{Enabled: true, Key: "a", Value: "1"},
		{Enabled: false, Key: "b", Value: "2"},
		{Enabled: true},
	}}
	got := s.Enabled()
	if len(got) != 1 || got[0].Key != "a" {
		t.Errorf("Enabled() = %+v, want only the enabled row with a key", got)
	}
}
