package diversity

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

const usageLine = "usage: diversity [-h] [--max-shared N] [--max-popular F] [--baseline NAMES] [--min-distance N] themes [themes ...]"

// Main runs the diversity command on args (without the program name) and returns its
// exit status: 0 OK, 1 problems, 2 usage or input errors. Options parse as Python's
// argparse does: before or after the themes, "--opt value" or "--opt=value", any
// unambiguous prefix, and "--" ends the options.
func Main(args []string, usage string, stdout, stderr io.Writer) int {
	o := Options{MaxShared: 9, MaxPopular: 0.5, MinDistance: 8}
	baseline, baselineSet := "ink,charm", false
	names := []string{"--max-shared", "--max-popular", "--baseline", "--min-distance", "--help"}
	bad := func(format string, a ...any) int {
		fmt.Fprintln(stderr, usageLine)
		fmt.Fprintf(stderr, "diversity: error: "+format+"\n", a...)
		return 2
	}
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}
		if a == "-h" {
			fmt.Fprintln(stdout, usage)
			return 0
		}
		if !strings.HasPrefix(a, "-") || a == "-" || isNumber(a) {
			paths = append(paths, a)
			continue
		}
		key, val, hasVal := strings.Cut(a, "=")
		var match []string
		for _, n := range names {
			if n == key {
				match = []string{n}
				break
			}
			if strings.HasPrefix(n, key) && strings.HasPrefix(key, "--") {
				match = append(match, n)
			}
		}
		switch len(match) {
		case 0:
			return bad("unrecognized arguments: %s", a)
		case 1:
		default:
			return bad("ambiguous option: %s could match %s", key, strings.Join(match, ", "))
		}
		if match[0] == "--help" {
			fmt.Fprintln(stdout, usage)
			return 0
		}
		if !hasVal {
			if i+1 >= len(args) || (strings.HasPrefix(args[i+1], "-") && !isNumber(args[i+1])) {
				return bad("argument %s: expected one argument", match[0])
			}
			i++
			val = args[i]
		}
		switch match[0] {
		case "--max-shared", "--min-distance":
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil {
				return bad("argument %s: invalid int value: '%s'", match[0], val)
			}
			if match[0] == "--max-shared" {
				o.MaxShared = n
			} else {
				o.MinDistance = n
			}
		case "--max-popular":
			f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
			if err != nil {
				return bad("argument --max-popular: invalid float value: '%s'", val)
			}
			o.MaxPopular = f
		case "--baseline":
			baseline, baselineSet = val, true
		}
	}
	if len(paths) == 0 {
		return bad("the following arguments are required: themes")
	}
	o.Baseline = map[string]bool{}
	for _, n := range strings.Split(baseline, ",") {
		o.Baseline[n] = true
	}
	themes := make([]Theme, 0, len(paths))
	for _, p := range paths {
		t, err := Load(p)
		if err != nil {
			fmt.Fprintln(stderr, "diversity:", err)
			return 2
		}
		themes = append(themes, t)
	}
	if missing := MissingBaselines(themes, baseline); len(missing) > 0 {
		if baselineSet {
			fmt.Fprintln(stderr, "diversity:", missingError(missing))
			return 2
		}
		// The default baseline is advisory, so comparing brand themes among themselves
		// still works; an explicit --baseline is a request, so a missing one fails above.
		fmt.Fprintf(stderr, "diversity: note: %s, so the min-distance rule did not run against them (pass %s, or --baseline \"\" to silence this)\n",
			notAmong(missing), kitPaths(missing))
	}
	out, code := Report(themes, o)
	io.WriteString(stdout, out)
	return code
}

// MissingBaselines returns the names in a --baseline list that no theme carries, in
// list order. Empty names ("--baseline ”") ask for no baseline and are never missing.
// A missing baseline turns the min-distance rule off without a word, which is how
// "the default with a new palette" got through, so the CLI fails on one.
func MissingBaselines(themes []Theme, baseline string) []string {
	have := map[string]bool{}
	for _, t := range themes {
		have[t.Name] = true
	}
	var missing []string
	seen := map[string]bool{}
	for _, n := range strings.Split(baseline, ",") {
		if n != "" && !have[n] && !seen[n] {
			missing = append(missing, n)
			seen[n] = true
		}
	}
	return missing
}

func missingError(missing []string) string {
	files, it := "its theme file", "it"
	if len(missing) > 1 {
		files, it = "their theme files", "them"
	}
	return fmt.Sprintf("%s, so the min-distance rule cannot run: pass %s too (e.g. %s, from the skill directory) or drop %s from --baseline",
		notAmong(missing), files, kitPaths(missing), it)
}

func notAmong(missing []string) string {
	q := make([]string, len(missing))
	for i, n := range missing {
		q[i] = strconv.Quote(n)
	}
	if len(missing) == 1 {
		return "baseline " + q[0] + " is not among the themes"
	}
	return "baselines " + strings.Join(q, ", ") + " are not among the themes"
}

func kitPaths(names []string) string {
	p := make([]string, len(names))
	for i, n := range names {
		p[i] = "kit/themes/" + n + ".json"
	}
	return strings.Join(p, " ")
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
