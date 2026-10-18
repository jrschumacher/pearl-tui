package feedback

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// URLLimit is the longest new-issue URL submit prints. GitHub and browsers accept a little
// more, but a URL over ~8 KB fails in some of them with a 414 and no form at all.
const URLLimit = 8000

// noResponse is what an issue form writes for a field left empty.
const noResponse = "_No response_"

// Issue is one draft: where it goes, which form, and the value of each field.
type Issue struct {
	Repo    string
	Kind    Kind
	Title   string
	Values  map[string]string // by field id
	Comment int               // > 0: post the body as a comment on this issue instead
}

// Missing lists the required fields with no value. A comment adds evidence to an issue
// that already holds the rest, so it needs only one field that isn't filled in for it.
func (is Issue) Missing() []string {
	var out []string
	if is.Comment > 0 {
		for _, f := range is.Kind.Fields {
			if !f.Auto && strings.TrimSpace(is.Values[f.ID]) != "" {
				return nil
			}
		}
		return []string{"any field (a comment needs one)"}
	}
	for _, f := range is.Kind.Fields {
		if f.Required && strings.TrimSpace(is.Values[f.ID]) == "" {
			out = append(out, f.ID)
		}
	}
	if strings.TrimSpace(is.Title) == "" && is.Comment == 0 {
		out = append([]string{"title"}, out...)
	}
	return out
}

// Body renders the issue the way GitHub renders a submitted issue form: a "### Label"
// heading per field, its value (fenced when the form renders it as code), and
// "_No response_" for an empty one. A comment leaves the empty fields out.
func (is Issue) Body() string {
	var b strings.Builder
	for _, f := range is.Kind.Fields {
		v := strings.TrimSpace(is.Values[f.ID])
		if v == "" && is.Comment > 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "### %s\n\n", f.Label)
		switch {
		case v == "":
			b.WriteString(noResponse)
		case f.Render != "":
			fmt.Fprintf(&b, "```%s\n%s\n```", f.Render, v)
		default:
			b.WriteString(v)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Header is the part of a draft that is not the body: where it goes and how it's labelled.
func (is Issue) Header() string {
	var b strings.Builder
	fmt.Fprintf(&b, "repo:     %s\n", is.Repo)
	if is.Comment > 0 {
		fmt.Fprintf(&b, "comment:  #%d\n", is.Comment)
		return b.String()
	}
	fmt.Fprintf(&b, "template: %s\n", is.Kind.Template)
	fmt.Fprintf(&b, "labels:   %s\n", strings.Join(is.Kind.Labels, ", "))
	fmt.Fprintf(&b, "title:    %s\n", is.Title)
	return b.String()
}

// Draft is the full text `feedback draft` prints: the header, a rule, the body.
func (is Issue) Draft() string {
	return is.Header() + "\n" + is.Body()
}

// NewIssueURL builds the new-issue link that opens the form with the title, labels and
// fields filled in, keeping it within limit bytes. Fields are added in form order while
// they fit; the ones that don't, and dropdowns (GitHub prefills only inputs and
// textareas), are returned as skipped, to be pasted or picked by hand.
func (is Issue) NewIssueURL(limit int) (link string, skipped []string) {
	base := "https://github.com/" + is.Repo + "/issues/new?"
	q := []string{
		"template=" + url.QueryEscape(is.Kind.Template),
		"title=" + url.QueryEscape(is.Title),
		"labels=" + url.QueryEscape(strings.Join(is.Kind.Labels, ",")),
	}
	link = base + strings.Join(q, "&")
	for _, f := range is.Kind.Fields {
		v := strings.TrimSpace(is.Values[f.ID])
		if v == "" {
			continue
		}
		if f.Type == "dropdown" {
			skipped = append(skipped, f.ID)
			continue
		}
		p := "&" + url.QueryEscape(f.ID) + "=" + url.QueryEscape(v)
		if len(link)+len(p) > limit {
			skipped = append(skipped, f.ID)
			continue
		}
		link += p
	}
	return link, skipped
}

// IssueURL is the page of an existing issue, for a comment filed by hand.
func IssueURL(repo string, n int) string {
	return fmt.Sprintf("https://github.com/%s/issues/%d", repo, n)
}

// SearchURL is the repository's issue search for terms, open and closed.
func SearchURL(repo string, terms []string) string {
	return "https://github.com/" + repo + "/issues?q=" + url.QueryEscape("is:issue "+strings.Join(terms, " "))
}

var stopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "when": true, "that": true, "this": true,
	"from": true, "into": true, "not": true, "but": true, "are": true, "was": true, "its": true,
	"has": true, "have": true, "does": true, "doesn": true, "don": true, "isn": true, "can": true,
	"should": true, "would": true, "under": true, "over": true, "only": true, "every": true,
	"bug": true, "fix": true, "feat": true, "skill": true,
}

// Terms picks a title's search words: lowercased, punctuation dropped, stopwords and words
// under three letters out, each once, at most six (GitHub search ANDs them, so fewer
// words find more).
func Terms(title string) []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.')
	}) {
		w = strings.Trim(w, "-_.")
		if len(w) < 3 || stopwords[w] || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	if len(out) > 6 {
		// keep the longest: they carry the most meaning ("contrast", "dialog"), in title order
		idx := make([]int, len(out))
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(a, b int) bool { return len(out[idx[a]]) > len(out[idx[b]]) })
		idx = idx[:6]
		sort.Ints(idx)
		kept := make([]string, 0, 6)
		for _, i := range idx {
			kept = append(kept, out[i])
		}
		out = kept
	}
	return out
}
