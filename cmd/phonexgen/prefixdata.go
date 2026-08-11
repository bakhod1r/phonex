package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The geocoding, carrier and time zone data sets are plain text: one
// "prefix|value" line per entry, with '#' comments, split across a file per
// calling code (the time zone data is a single file). Values repeat heavily —
// tens of thousands of English place names collapse to a few thousand — so
// the generated table stores each distinct value once and refers to it by
// index.

// prefixEntry is one parsed line.
type prefixEntry struct {
	prefix uint32
	length int
	value  string
}

// prefixDataset is a parsed data set, ready to emit.
type prefixDataset struct {
	entries []prefixEntry
	// table holds the distinct values in emission order.
	table []string
	// index maps a value to its position in table.
	index          map[string]uint32
	minLen, maxLen int
	sourceHash     [sha256.Size]byte
}

// readPrefixData reads every .txt file under dir and returns the data set.
// Files are read in sorted order so that the output is reproducible.
func readPrefixData(dir string) (*prefixDataset, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no .txt files under %s", dir)
	}
	sort.Strings(names)

	d := &prefixDataset{index: map[string]uint32{}, minLen: 1 << 30}
	hash := sha256.New()

	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		// The hash covers the file names as well as their contents, so that
		// adding or removing a language file is detected too.
		hash.Write([]byte(filepath.Base(name)))
		hash.Write(raw)

		scanner := bufio.NewScanner(bytes.NewReader(raw))
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		line := 0
		for scanner.Scan() {
			line++
			text := strings.TrimSpace(scanner.Text())
			if text == "" || strings.HasPrefix(text, "#") {
				continue
			}
			bar := strings.IndexByte(text, '|')
			if bar < 0 {
				return nil, fmt.Errorf("%s:%d: no '|' separator", name, line)
			}
			prefix, value := text[:bar], text[bar+1:]
			if value == "" {
				// Upstream uses an empty value to say "no description here",
				// which is the same as having no entry at all.
				continue
			}
			if len(prefix) > 9 {
				return nil, fmt.Errorf("%s:%d: prefix %q is longer than 9 digits", name, line, prefix)
			}
			v, err := strconv.ParseUint(prefix, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: prefix %q: %w", name, line, prefix, err)
			}
			if prefix[0] == '0' {
				return nil, fmt.Errorf("%s:%d: prefix %q has a leading zero", name, line, prefix)
			}
			d.entries = append(d.entries, prefixEntry{
				prefix: uint32(v),
				length: len(prefix),
				value:  value,
			})
			if len(prefix) < d.minLen {
				d.minLen = len(prefix)
			}
			if len(prefix) > d.maxLen {
				d.maxLen = len(prefix)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	if len(d.entries) == 0 {
		return nil, fmt.Errorf("no entries under %s", dir)
	}
	copy(d.sourceHash[:], hash.Sum(nil))

	sort.Slice(d.entries, func(i, j int) bool { return d.entries[i].prefix < d.entries[j].prefix })
	for i := 1; i < len(d.entries); i++ {
		if d.entries[i].prefix == d.entries[i-1].prefix {
			return nil, fmt.Errorf("prefix %d appears twice", d.entries[i].prefix)
		}
	}

	// Build the value table in order of first appearance, which keeps the
	// generated file stable across runs.
	for _, e := range d.entries {
		if _, ok := d.index[e.value]; !ok {
			d.index[e.value] = uint32(len(d.table))
			d.table = append(d.table, e.value)
		}
	}
	return d, nil
}

// writePrefixMap emits the shared lookup table. valueName is the identifier
// the calling package uses for the value table, and is emitted separately by
// the caller so that each package can choose its own value type.
func (d *prefixDataset) writePrefixMap(b *bytes.Buffer) {
	b.WriteString("// numbers is the prefix lookup table. Prefixes are stored as their\n")
	b.WriteString("// numeric value, in ascending order, alongside the index of the value\n")
	b.WriteString("// each one maps to.\n")
	b.WriteString("var numbers = prefixmap.Map{\n")
	fmt.Fprintf(b, "\tMinLen: %d,\n\tMaxLen: %d,\n", d.minLen, d.maxLen)

	b.WriteString("\tPrefixes: []uint32{\n")
	writeUint32s(b, func(i int) uint32 { return d.entries[i].prefix }, len(d.entries))
	b.WriteString("\t},\n")

	b.WriteString("\tValues: []uint32{\n")
	writeUint32s(b, func(i int) uint32 { return d.index[d.entries[i].value] }, len(d.entries))
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")
}

// writeUint32s emits n values, wrapped so that no source line grows
// unreasonably long.
func writeUint32s(b *bytes.Buffer, at func(int) uint32, n int) {
	const perLine = 16
	for i := 0; i < n; i++ {
		if i%perLine == 0 {
			b.WriteString("\t\t")
		}
		b.WriteString(strconv.FormatUint(uint64(at(i)), 10))
		b.WriteByte(',')
		if i%perLine == perLine-1 || i == n-1 {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
	}
}

// generatePrefixPackage writes a package whose value table is a []string.
func generatePrefixPackage(dir, outPath, pkg, doc string) error {
	d, err := readPrefixData(dir)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by phonexgen from libphonenumber %s. DO NOT EDIT.\n\n", doc)
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	b.WriteString("import \"github.com/bakhod1r/phonex/internal/prefixmap\"\n\n")
	fmt.Fprintf(&b, "// SourceHash is the SHA-256 of the %s data this file was generated from.\n", doc)
	fmt.Fprintf(&b, "const SourceHash = %q\n\n", hex.EncodeToString(d.sourceHash[:]))

	d.writePrefixMap(&b)

	b.WriteString("// values holds each distinct value once; the lookup table refers to\n")
	b.WriteString("// them by index.\n")
	b.WriteString("var values = []string{\n")
	for _, v := range d.table {
		fmt.Fprintf(&b, "\t%q,\n", v)
	}
	b.WriteString("}\n")

	return writeGenerated(outPath, b.Bytes(), fmt.Sprintf(
		"%d entries, %d distinct values (source %s)",
		len(d.entries), len(d.table), hex.EncodeToString(d.sourceHash[:8])))
}

// generateTimezonePackage writes the time zone package, whose values are
// lists of zone names rather than single strings.
func generateTimezonePackage(dir, outPath string) error {
	d, err := readPrefixData(dir)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	b.WriteString("// Code generated by phonexgen from libphonenumber timezones/map_data.txt. DO NOT EDIT.\n\n")
	b.WriteString("package timezone\n\n")
	b.WriteString("import \"github.com/bakhod1r/phonex/internal/prefixmap\"\n\n")
	b.WriteString("// SourceHash is the SHA-256 of the time zone data this file was\n")
	b.WriteString("// generated from.\n")
	fmt.Fprintf(&b, "const SourceHash = %q\n\n", hex.EncodeToString(d.sourceHash[:]))

	d.writePrefixMap(&b)

	b.WriteString("// values holds each distinct set of zones once. Upstream separates the\n")
	b.WriteString("// zones of one entry with '&'.\n")
	b.WriteString("var values = [][]string{\n")
	for _, v := range d.table {
		zones := strings.Split(v, "&")
		quoted := make([]string, len(zones))
		for i, z := range zones {
			quoted[i] = strconv.Quote(strings.TrimSpace(z))
		}
		fmt.Fprintf(&b, "\t{%s},\n", strings.Join(quoted, ", "))
	}
	b.WriteString("}\n")

	return writeGenerated(outPath, b.Bytes(), fmt.Sprintf(
		"%d entries, %d distinct zone sets (source %s)",
		len(d.entries), len(d.table), hex.EncodeToString(d.sourceHash[:8])))
}

// writeGenerated formats and writes a generated file.
func writeGenerated(outPath string, src []byte, summary string) error {
	formatted, err := format.Source(src)
	if err != nil {
		_ = os.WriteFile(outPath+".broken", src, 0o644)
		return fmt.Errorf("formatting %s: %w (raw output written to %s.broken)", outPath, err, outPath)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %s\n", outPath, summary)
	return nil
}
