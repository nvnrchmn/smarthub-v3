package notify

import "testing"

func TestNormalisasiNomor(t *testing.T) {
	kasus := map[string]string{
		"0898-3342-429":     "628983342429",
		"08983342429":       "628983342429",
		"+62 812 3456 789":  "628123456789",
		"6281234567890":     "6281234567890",
		"8123456789":        "628123456789",
		"  0812 3456 789 ":  "628123456789",
		"":                  "",
		"abc":               "",
		"62-812-3456-7890":  "6281234567890",
		"008123456789":      "628123456789",
	}
	for masuk, harap := range kasus {
		if dapat := NormalisasiNomor(masuk); dapat != harap {
			t.Errorf("NormalisasiNomor(%q) = %q, harap %q", masuk, dapat, harap)
		}
	}
}
