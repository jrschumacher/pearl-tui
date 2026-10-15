package cellshot

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Main is the cellshot command: argument handling, one report line per input, and the
// PNG, sheet or GIF output. usage is printed for -h and for bad arguments. It returns
// the exit status: 1 when --cols/--rows do not match, 2 for usage or I/O errors.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer, usage string) int {
	opts := map[string]string{"cell": "12x24", "bg": "#16161a", "fg": "#d8d8d8", "ms": "100", "scale": "1"}
	flags := map[string]bool{}
	var files []string
	valued := []string{"--sheet", "--gif", "--cell", "--bg", "--fg", "--cols", "--rows", "--ms", "--scale", "--label"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case slices.Contains(valued, a):
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "cellshot: %s needs a value\n", a)
				return 2
			}
			opts[a[2:]] = args[i+1]
			i++
		case a == "--grid" || a == "--light":
			flags[a[2:]] = true
		case a == "-h" || a == "--help":
			fmt.Fprint(stdout, usage)
			return 0
		default:
			files = append(files, a)
		}
	}
	if flags["light"] {
		opts["bg"], opts["fg"] = "#fafafa", "#222222"
	}
	out := opts["sheet"]
	if out == "" {
		out = opts["gif"]
	}
	if out == "" {
		if len(files) != 2 {
			fmt.Fprint(stdout, usage)
			return 2
		}
		files, out = files[:1], files[1]
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "cellshot: no input frames")
		return 2
	}
	fail := func(err error) int {
		fmt.Fprintf(stderr, "cellshot: %v\n", err)
		return 2
	}
	var o Options
	if _, err := fmt.Sscanf(opts["cell"], "%dx%d", &o.CellW, &o.CellH); err != nil || o.CellW < 1 || o.CellH < 1 {
		return fail(fmt.Errorf("bad --cell %q (want WxH, e.g. 12x24)", opts["cell"]))
	}
	var err error
	if o.BG, err = ParseHex(opts["bg"]); err != nil {
		return fail(err)
	}
	if o.FG, err = ParseHex(opts["fg"]); err != nil {
		return fail(err)
	}
	var bad error
	num := func(k string) int {
		v, ok := opts[k]
		if !ok {
			return 0
		}
		n, err := strconv.Atoi(v)
		if err != nil && bad == nil {
			bad = fmt.Errorf("bad --%s %q (want an integer)", k, v)
		}
		return n
	}
	ms, scale := num("ms"), num("scale")
	o.Cols, o.Rows = num("cols"), num("rows")
	if bad != nil {
		return fail(bad)
	}
	o.Grid = flags["grid"]

	status := 0
	var images []*image.RGBA
	for _, f := range files {
		var data []byte
		if f == "-" {
			data, err = io.ReadAll(stdin)
		} else {
			data, err = os.ReadFile(f)
		}
		if err != nil {
			return fail(err)
		}
		text := strings.TrimSuffix(string(data), "\n")
		rows, warns := Parse(text)
		problems := append([]string(nil), warns...)
		maxW := 0
		for _, r := range rows {
			maxW = max(maxW, len(r))
		}
		var bad []int
		for k, r := range rows {
			if len(r) != maxW {
				bad = append(bad, k)
			}
		}
		if len(bad) > 0 {
			problems = append(problems, fmt.Sprintf("rows of unequal width: %d rows short of %d (first: row %d = %d)",
				len(bad), maxW, bad[0], len(rows[bad[0]])))
		}
		if o.Cols != 0 && len(rows) > 0 && maxW != o.Cols {
			problems = append(problems, fmt.Sprintf("width %d != expected %d", maxW, o.Cols))
			status = 1
		}
		if o.Rows != 0 && len(rows) != o.Rows {
			problems = append(problems, fmt.Sprintf("height %d != expected %d", len(rows), o.Rows))
			status = 1
		}
		img, w2 := Render(rows, o)
		problems = append(problems, w2...)
		img = Scale(img, scale)
		label := opts["label"]
		if label == "" && opts["sheet"] != "" {
			label = stem(f)
		}
		if label != "" {
			img = Label(img, label, o.BG)
		}
		images = append(images, img)
		line := fmt.Sprintf("%s: %dx%d cells", f, maxW, len(rows))
		if len(problems) > 0 {
			line += "\n  - " + strings.Join(problems, "\n  - ")
		}
		fmt.Fprintln(stdout, line)
	}
	if err := write(out, images, opts, ms); err != nil {
		return fail(err)
	}
	fmt.Fprintf(stdout, "wrote %s\n", out)
	return status
}

// stem is Python's os.path.splitext(os.path.basename(f))[0].
func stem(f string) string {
	b := filepath.Base(f)
	if f == "-" {
		return "-"
	}
	lead := len(b) - len(strings.TrimLeft(b, "."))
	if i := strings.LastIndexByte(b[lead:], '.'); i > 0 {
		return b[:lead+i]
	}
	return b
}

func write(out string, images []*image.RGBA, opts map[string]string, ms int) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	switch {
	case opts["gif"] != "":
		err = WriteGIF(f, images, ms)
	case opts["sheet"] != "":
		err = encodePNG(f, Sheet(images))
	default:
		err = encodePNG(f, images[0])
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// encodePNG writes img as PNG. Every pixel is opaque, so it is stored as 8-bit RGB.
func encodePNG(w io.Writer, img *image.RGBA) error {
	return (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(w, img)
}
