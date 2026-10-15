package pyfmt

import "testing"

func TestStr(t *testing.T) {
	for in, want := range map[string]string{
		`"x"`: "x", `true`: "True", `false`: "False", `null`: "None", `3`: "3", `-0`: "0",
		`1.0`: "1.0", `1e16`: "1e+16", `1e-5`: "1e-05", `0.0001`: "0.0001", `2.50`: "2.5",
		`["a", 1, "it's"]`: `['a', 1, "it's"]`, `{"b": 1, "a": null}`: `{'a': None, 'b': 1}`,
	} {
		v, err := Decode([]byte(in))
		if err != nil {
			t.Fatal(err)
		}
		if got := Str(v); got != want {
			t.Errorf("Str(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestTruthy(t *testing.T) {
	for in, want := range map[string]bool{`""`: false, `"x"`: true, `0`: false, `0.0`: false, `2`: true,
		`null`: false, `true`: true, `[]`: false, `{}`: false, `[0]`: true} {
		v, _ := Decode([]byte(in))
		if got := Truthy(v); got != want {
			t.Errorf("Truthy(%s) = %v", in, got)
		}
	}
}

func TestKeysKeepFirstPosition(t *testing.T) {
	keys, err := Keys([]byte(`{"light": {}, "dark": 1, "light": 2, "note": "x"}`))
	if err != nil || len(keys) != 3 || keys[0] != "light" || keys[1] != "dark" || keys[2] != "note" {
		t.Errorf("Keys = %v, %v", keys, err)
	}
}

func TestQuoteJSON(t *testing.T) {
	if got, want := QuoteJSON("Cr\u00e8me \u2615 \"q\"\n\x7f\U0001F600"), `"Cr\u00e8me \u2615 \"q\"\n\u007f\ud83d\ude00"`; got != want {
		t.Errorf("QuoteJSON = %s, want %s", got, want)
	}
}

func TestPadding(t *testing.T) {
	if got := Ljust(Head("muted-upper", 10), 12); got != "muted-uppe  " {
		t.Errorf("%q", got)
	}
	if got := Ljust(Head("≈≈", 10), 4); got != "≈≈  " {
		t.Errorf("%q", got)
	}
}
