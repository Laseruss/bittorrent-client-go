package bencode

import (
	"reflect"
	"strings"
	"testing"
)

func TestDecodeNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{input: "i43e", expected: 43},
		{input: "i-43e", expected: -43},
		{input: "i539857e", expected: 539857},
		{input: "i5e", expected: 5},
	}

	for _, tt := range tests {
		d := NewDecoder(strings.NewReader(tt.input))
		got, err := d.decodeNumber()
		if err != nil {
			t.Fatalf("expected to be able to decode into number, %s", err)
		}

		if got != tt.expected {
			t.Fatalf("expected number to be %d, got=%d", tt.expected, got)
		}
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "4:spam", expected: "spam"},
		{input: "5:HelLo", expected: "HelLo"},
		{input: "0:", expected: ""},
	}

	for _, tt := range tests {
		d := NewDecoder(strings.NewReader(tt.input))
		got, err := d.decodeString()
		if err != nil {
			t.Fatalf("expected to be able to decode into string, %s", err)
		}

		if string(got) != tt.expected {
			t.Fatalf("expected string to be %s, got=%s", tt.expected, got)
		}
	}
}

func TestDecodeList(t *testing.T) {
	tests := []struct {
		input    string
		expected List
	}{
		{
			input:    "l4:spami34ee",
			expected: List{"spam", 34},
		},
		{
			input:    "li-45e5:helloe",
			expected: List{-45, "hello"},
		},
	}

	for _, tt := range tests {
		d := NewDecoder(strings.NewReader(tt.input))
		val, err := d.decodeVal()
		if err != nil {
			t.Fatalf("could not decode value list: %s\n", err)
		}

		l, ok := val.(List)
		if !ok {
			t.Fatalf("expected val to be a list, got=%T", val)
		}

		for i, val := range l {
			if !reflect.DeepEqual(val, tt.expected[i]) {
				t.Fatalf("expected and value doesn't match, wanted=%v, got=%v", val, tt.expected[i])
			}
		}
	}
}

func TestDecodeDictionary(t *testing.T) {
	tests := []struct {
		input    string
		expected Dictionary
	}{
		{
			input: "d1:ai5e1:bl4:spam5:helloee",
			expected: Dictionary{
				"a": 5,
				"b": List{"spam", "hello"},
			},
		},
		{
			input:    "d6:myDictd1:ai10eee",
			expected: Dictionary{"myDict": Dictionary{"a": 10}},
		},
	}

	for _, tt := range tests {
		d := NewDecoder(strings.NewReader(tt.input))
		val, err := d.decodeVal()
		if err != nil {
			t.Fatalf("could not decode value dictionary: %s\n", err)
		}

		dict, ok := val.(Dictionary)
		if !ok {
			t.Fatalf("expected val to be a dictionary, got=%T", val)
		}

		if !reflect.DeepEqual(dict, tt.expected) {
			t.Fatalf("expected and value doesn't match, wanted=%v, got=%v", tt.expected, val)
		}
	}
}
