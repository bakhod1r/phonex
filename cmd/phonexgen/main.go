// Command phonexgen generates countries/generated.go from Google's
// libphonenumber PhoneNumberMetadata.xml.
//
// Usage:
//
//	go run ./cmd/phonexgen \
//	    -xml internal/metadata/PhoneNumberMetadata.xml \
//	    -regions internal/metadata/metadata.json \
//	    -out countries/generated.go
//
// The XML is the upstream file, vendored verbatim. The JSON supplies the
// region details libphonenumber does not carry (ISO-3166 alpha-3 code,
// English country name, IANA time zones).
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("phonexgen: ")

	xmlPath := flag.String("xml", "internal/metadata/PhoneNumberMetadata.xml", "path to libphonenumber PhoneNumberMetadata.xml")
	regionsPath := flag.String("regions", "internal/metadata/metadata.json", "path to region detail JSON (ISO3, name, timezones)")
	outPath := flag.String("out", "countries/generated.go", "path of the generated Go file")
	shortXMLPath := flag.String("short-xml", "internal/metadata/ShortNumberMetadata.xml", "path to libphonenumber ShortNumberMetadata.xml")
	shortOutPath := flag.String("short-out", "shortnumber/generated.go", "path of the generated short number file")
	tzDir := flag.String("timezone-dir", "internal/metadata/timezones", "directory holding the time zone data")
	tzOut := flag.String("timezone-out", "timezone/generated.go", "path of the generated time zone file")
	carrierDir := flag.String("carrier-dir", "internal/metadata/carrier/en", "directory holding the carrier data")
	carrierOut := flag.String("carrier-out", "carrier/generated.go", "path of the generated carrier file")
	geoDir := flag.String("geocoding-dir", "internal/metadata/geocoding/en", "directory holding the geocoding data")
	geoOut := flag.String("geocoding-out", "geocoding/generated.go", "path of the generated geocoding file")
	flag.Parse()

	if err := generateShort(*shortXMLPath, *shortOutPath); err != nil {
		log.Fatal(err)
	}
	if err := generateTimezonePackage(*tzDir, *tzOut); err != nil {
		log.Fatal(err)
	}
	if err := generatePrefixPackage(*carrierDir, *carrierOut, "carrier", "carrier/en"); err != nil {
		log.Fatal(err)
	}
	if err := generatePrefixPackage(*geoDir, *geoOut, "geocoding", "geocoding/en"); err != nil {
		log.Fatal(err)
	}

	raw, err := os.ReadFile(*xmlPath)
	if err != nil {
		log.Fatal(err)
	}
	var doc xmlDocument
	if err := xml.Unmarshal(raw, &doc); err != nil {
		log.Fatalf("parsing %s: %v", *xmlPath, err)
	}

	regions, err := loadRegions(*regionsPath)
	if err != nil {
		log.Fatal(err)
	}

	terrs := make([]*territory, 0, len(doc.Territories))
	for i := range doc.Territories {
		t, err := convert(&doc.Territories[i], regions)
		if err != nil {
			log.Fatalf("territory %s: %v", doc.Territories[i].ID, err)
		}
		terrs = append(terrs, t)
	}
	// Upstream only marks the main region where a calling code is shared;
	// a code used by a single region has no marker and that region owns it.
	shared := map[string]int{}
	for _, t := range terrs {
		shared[t.DialCode]++
	}
	for _, t := range terrs {
		if shared[t.DialCode] == 1 {
			t.IsMainCountry = true
		}
	}

	sort.Slice(terrs, func(i, j int) bool {
		if terrs[i].Key != terrs[j].Key {
			return terrs[i].Key < terrs[j].Key
		}
		return terrs[i].DialCode < terrs[j].DialCode
	})

	sourceHash := sha256.Sum256(raw)
	src, err := render(terrs, sourceHash)
	if err != nil {
		log.Fatal(err)
	}
	formatted, err := format.Source(src)
	if err != nil {
		// Write the unformatted source so the syntax error can be inspected.
		_ = os.WriteFile(*outPath+".broken", src, 0o644)
		log.Fatalf("formatting generated source: %v (raw output written to %s.broken)", err, *outPath)
	}
	if err := os.WriteFile(*outPath, formatted, 0o644); err != nil {
		log.Fatal(err)
	}

	geo, nonGeo := 0, 0
	for _, t := range terrs {
		if t.NonGeo {
			nonGeo++
		} else {
			geo++
		}
	}
	fmt.Printf("wrote %s: %d regions, %d non-geographical entities (source %s)\n",
		*outPath, geo, nonGeo, hex.EncodeToString(sourceHash[:8]))
}

// ---------------------------------------------------------------- XML schema

type xmlDocument struct {
	Territories []xmlTerritory `xml:"territories>territory"`
}

type xmlTerritory struct {
	ID                           string `xml:"id,attr"`
	CountryCode                  string `xml:"countryCode,attr"`
	MainCountryForCode           string `xml:"mainCountryForCode,attr"`
	LeadingDigits                string `xml:"leadingDigits,attr"`
	NationalPrefix               string `xml:"nationalPrefix,attr"`
	NationalPrefixForParsing     string `xml:"nationalPrefixForParsing,attr"`
	NationalPrefixTransformRule  string `xml:"nationalPrefixTransformRule,attr"`
	InternationalPrefix          string `xml:"internationalPrefix,attr"`
	PreferredInternationalPrefix string `xml:"preferredInternationalPrefix,attr"`
	PreferredExtnPrefix          string `xml:"preferredExtnPrefix,attr"`
	MobileNumberPortableRegion   string `xml:"mobileNumberPortableRegion,attr"`

	Formats []xmlNumberFormat `xml:"availableFormats>numberFormat"`

	General        *xmlDesc `xml:"generalDesc"`
	FixedLine      *xmlDesc `xml:"fixedLine"`
	Mobile         *xmlDesc `xml:"mobile"`
	TollFree       *xmlDesc `xml:"tollFree"`
	PremiumRate    *xmlDesc `xml:"premiumRate"`
	SharedCost     *xmlDesc `xml:"sharedCost"`
	VoIP           *xmlDesc `xml:"voip"`
	PersonalNumber *xmlDesc `xml:"personalNumber"`
	Pager          *xmlDesc `xml:"pager"`
	UAN            *xmlDesc `xml:"uan"`
	Voicemail      *xmlDesc `xml:"voicemail"`
	NoIntlDialling *xmlDesc `xml:"noInternationalDialling"`
}

type xmlDesc struct {
	Pattern         string          `xml:"nationalNumberPattern"`
	PossibleLengths *xmlPossibleLen `xml:"possibleLengths"`
	Example         string          `xml:"exampleNumber"`
}

type xmlPossibleLen struct {
	National  string `xml:"national,attr"`
	LocalOnly string `xml:"localOnly,attr"`
}

type xmlNumberFormat struct {
	Pattern                              string   `xml:"pattern,attr"`
	NationalPrefixFormattingRule         string   `xml:"nationalPrefixFormattingRule,attr"`
	NationalPrefixOptionalWhenFormatting string   `xml:"nationalPrefixOptionalWhenFormatting,attr"`
	CarrierCodeFormattingRule            string   `xml:"carrierCodeFormattingRule,attr"`
	LeadingDigits                        []string `xml:"leadingDigits"`
	Format                               string   `xml:"format"`
	IntlFormat                           []string `xml:"intlFormat"`
}

// ------------------------------------------------------------- region detail

type regionDetail struct {
	ISO3      string   `json:"iso3"`
	Name      string   `json:"name"`
	Timezones []string `json:"timezones"`
}

func loadRegions(path string) (map[string]regionDetail, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]regionDetail
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return m, nil
}

// ------------------------------------------------------------ intermediate

type desc struct {
	Pattern   string
	Lengths   []int32
	LocalOnly []int32
	Example   string
}

func (d desc) empty() bool {
	return d.Pattern == "" && len(d.Lengths) == 0 && len(d.LocalOnly) == 0 && d.Example == ""
}

type numberFormat struct {
	Pattern                string
	Format                 string
	LeadingDigits          string
	NationalPrefixRule     string
	NationalPrefixOptional bool
	CarrierCodeRule        string
	IntlFormat             string
}

type territory struct {
	// Key is the map key in the generated file: the ISO-3166 alpha-2 code
	// for regions, or the calling code for non-geographical entities.
	Key    string
	NonGeo bool

	ISO2, ISO3, Name string
	DialCode         string
	IsMainCountry    bool
	LeadingDigits    string

	NationalPrefix              string
	SimpleNationalPrefix        string
	NationalPrefixForParsing    string
	NationalPrefixTransformRule string

	InternationalPrefix          string
	PreferredInternationalPrefix string
	PreferredExtnPrefix          string
	MobileNumberPortable         bool

	General        desc
	Descs          [numDescs]desc
	NoIntlDialling desc
	Formats        []numberFormat

	MinLength, MaxLength int
	Timezones            []string
}

// numDescs mirrors countries.NumDescs; the order must stay in sync.
const numDescs = 10

// descNames documents the slot order for readability in the generated file.
var descNames = [numDescs]string{
	"FixedLine", "Mobile", "TollFree", "PremiumRate", "SharedCost",
	"VoIP", "PersonalNumber", "Pager", "UAN", "Voicemail",
}

func convert(x *xmlTerritory, regions map[string]regionDetail) (*territory, error) {
	if x.CountryCode == "" {
		return nil, fmt.Errorf("missing countryCode")
	}
	t := &territory{
		DialCode:                     x.CountryCode,
		IsMainCountry:                x.MainCountryForCode == "true",
		LeadingDigits:                cleanPattern(x.LeadingDigits),
		NationalPrefix:               x.NationalPrefix,
		NationalPrefixTransformRule:  x.NationalPrefixTransformRule,
		InternationalPrefix:          cleanPattern(x.InternationalPrefix),
		PreferredInternationalPrefix: x.PreferredInternationalPrefix,
		PreferredExtnPrefix:          x.PreferredExtnPrefix,
		MobileNumberPortable:         x.MobileNumberPortableRegion == "true",
	}

	// "001" is libphonenumber's pseudo-region for non-geographical entities
	// (satellite, shared-cost and universal numbers). Several of them exist,
	// each with its own calling code, so key them by that code.
	if x.ID == "001" {
		t.NonGeo = true
		t.Key = x.CountryCode
		t.ISO2 = "001"
		t.ISO3 = "001"
		t.Name = "Non-geographical entity +" + x.CountryCode
		t.IsMainCountry = true
	} else {
		t.Key = x.ID
		t.ISO2 = x.ID
		if d, ok := regions[x.ID]; ok {
			t.ISO3, t.Name, t.Timezones = d.ISO3, d.Name, d.Timezones
		}
		if t.ISO3 == "" || t.ISO3 == x.ID {
			t.ISO3 = x.ID
		}
		if t.Name == "" || t.Name == x.ID {
			t.Name = x.ID
		}
	}

	// nationalPrefixForParsing defaults to the national prefix itself.
	npp := x.NationalPrefixForParsing
	if npp == "" {
		npp = regexp.QuoteMeta(x.NationalPrefix)
	}
	t.NationalPrefixForParsing = cleanPattern(npp)
	// When the prefix is a plain literal the parser can strip it with a
	// string comparison instead of a regexp match.
	if x.NationalPrefixForParsing == "" && x.NationalPrefixTransformRule == "" {
		t.SimpleNationalPrefix = x.NationalPrefix
	}

	descs := [numDescs]*xmlDesc{
		x.FixedLine, x.Mobile, x.TollFree, x.PremiumRate, x.SharedCost,
		x.VoIP, x.PersonalNumber, x.Pager, x.UAN, x.Voicemail,
	}
	for i, xd := range descs {
		d, err := convertDesc(xd)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", descNames[i], err)
		}
		t.Descs[i] = d
	}
	var err error
	if t.General, err = convertDesc(x.General); err != nil {
		return nil, fmt.Errorf("generalDesc: %w", err)
	}
	if t.NoIntlDialling, err = convertDesc(x.NoIntlDialling); err != nil {
		return nil, fmt.Errorf("noInternationalDialling: %w", err)
	}

	// The general descriptor never carries possibleLengths upstream; it is
	// the union of the per-type lengths.
	if len(t.General.Lengths) == 0 {
		t.General.Lengths = unionLengths(t.Descs[:], func(d desc) []int32 { return d.Lengths })
	}
	if len(t.General.LocalOnly) == 0 {
		t.General.LocalOnly = unionLengths(t.Descs[:], func(d desc) []int32 { return d.LocalOnly })
	}
	if n := len(t.General.Lengths); n > 0 {
		t.MinLength = int(t.General.Lengths[0])
		t.MaxLength = int(t.General.Lengths[n-1])
	}

	for i := range x.Formats {
		f, err := convertFormat(&x.Formats[i])
		if err != nil {
			return nil, fmt.Errorf("numberFormat %d: %w", i, err)
		}
		t.Formats = append(t.Formats, f)
	}
	return t, nil
}

func convertDesc(x *xmlDesc) (desc, error) {
	if x == nil {
		return desc{}, nil
	}
	d := desc{
		Pattern: cleanPattern(x.Pattern),
		Example: strings.TrimSpace(x.Example),
	}
	if x.PossibleLengths != nil {
		var err error
		if d.Lengths, err = parseLengths(x.PossibleLengths.National); err != nil {
			return desc{}, fmt.Errorf("possibleLengths national=%q: %w", x.PossibleLengths.National, err)
		}
		if d.LocalOnly, err = parseLengths(x.PossibleLengths.LocalOnly); err != nil {
			return desc{}, fmt.Errorf("possibleLengths localOnly=%q: %w", x.PossibleLengths.LocalOnly, err)
		}
	}
	return d, nil
}

func convertFormat(x *xmlNumberFormat) (numberFormat, error) {
	if x.Pattern == "" {
		return numberFormat{}, fmt.Errorf("missing pattern")
	}
	f := numberFormat{
		Pattern:                cleanPattern(x.Pattern),
		Format:                 strings.TrimSpace(x.Format),
		NationalPrefixRule:     strings.TrimSpace(x.NationalPrefixFormattingRule),
		NationalPrefixOptional: x.NationalPrefixOptionalWhenFormatting == "true",
		CarrierCodeRule:        strings.TrimSpace(x.CarrierCodeFormattingRule),
	}
	// Upstream lists leadingDigits from least to most specific; the last one
	// is the tightest test and implies the earlier ones.
	if n := len(x.LeadingDigits); n > 0 {
		f.LeadingDigits = cleanPattern(x.LeadingDigits[n-1])
	}
	if n := len(x.IntlFormat); n > 0 {
		f.IntlFormat = strings.TrimSpace(x.IntlFormat[n-1])
	}
	if f.Format == "" {
		return numberFormat{}, fmt.Errorf("missing format for pattern %q", f.Pattern)
	}
	return f, nil
}

// cleanPattern strips the whitespace upstream uses to keep the XML readable.
// The patterns are written so that removing all whitespace is safe.
func cleanPattern(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r':
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parseLengths reads a possibleLengths attribute such as "7,9-11" or "[4-8]"
// into a sorted, de-duplicated list of lengths.
func parseLengths(s string) ([]int32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	seen := map[int32]bool{}
	var out []int32
	for _, part := range strings.Split(s, ",") {
		// Upstream brackets ranges, e.g. national="[6-8],10,12".
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, "[")
		part = strings.TrimSuffix(part, "]")
		if part == "" {
			continue
		}
		lo, hi := part, part
		if i := strings.Index(part, "-"); i > 0 {
			lo, hi = part[:i], part[i+1:]
		}
		a, err := strconv.Atoi(strings.TrimSpace(lo))
		if err != nil {
			return nil, err
		}
		b, err := strconv.Atoi(strings.TrimSpace(hi))
		if err != nil {
			return nil, err
		}
		if a > b {
			return nil, fmt.Errorf("descending range %q", part)
		}
		for n := a; n <= b; n++ {
			if !seen[int32(n)] {
				seen[int32(n)] = true
				out = append(out, int32(n))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func unionLengths(ds []desc, pick func(desc) []int32) []int32 {
	seen := map[int32]bool{}
	var out []int32
	for _, d := range ds {
		for _, l := range pick(d) {
			if !seen[l] {
				seen[l] = true
				out = append(out, l)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ------------------------------------------------------------------ emitter

func render(terrs []*territory, sourceHash [sha256.Size]byte) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("// Code generated by phonexgen from libphonenumber PhoneNumberMetadata.xml. DO NOT EDIT.\n\n")
	b.WriteString("package countries\n\n")
	b.WriteString("import \"github.com/bakhod1r/phonex/internal/lazyre\"\n\n")
	b.WriteString("// SourceHash is the SHA-256 of the PhoneNumberMetadata.xml this file was\n")
	b.WriteString("// generated from. It identifies exactly which upstream revision of the\n")
	b.WriteString("// metadata is baked in, and lets a test confirm the vendored XML and the\n")
	b.WriteString("// generated table have not drifted apart.\n")
	fmt.Fprintf(&b, "const SourceHash = %q\n\n", hex.EncodeToString(sourceHash[:]))

	geo := make([]*territory, 0, len(terrs))
	nonGeo := make([]*territory, 0, 8)
	for _, t := range terrs {
		if t.NonGeo {
			nonGeo = append(nonGeo, t)
		} else {
			geo = append(geo, t)
		}
	}

	b.WriteString("// Data maps ISO-3166 alpha-2 region codes to their metadata.\n")
	b.WriteString("var Data = map[string]*Metadata{\n")
	for _, t := range geo {
		fmt.Fprintf(&b, "\t%q: %s,\n", t.Key, varName(t))
	}
	b.WriteString("}\n\n")

	b.WriteString("// NonGeo maps calling codes of non-geographical entities (libphonenumber\n")
	b.WriteString("// region \"001\") to their metadata.\n")
	b.WriteString("var NonGeo = map[string]*Metadata{\n")
	for _, t := range nonGeo {
		fmt.Fprintf(&b, "\t%q: %s,\n", t.Key, varName(t))
	}
	b.WriteString("}\n\n")

	if err := renderDialCodeIndex(&b, terrs); err != nil {
		return nil, err
	}

	for _, t := range terrs {
		renderTerritory(&b, t)
	}
	return b.Bytes(), nil
}

func varName(t *territory) string {
	if t.NonGeo {
		return "md001_" + t.DialCode
	}
	return "md" + strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, t.Key)
}

// renderDialCodeIndex emits fixed-size lookup tables keyed by the numeric
// value of a 1-, 2- or 3-digit calling code. Calling codes never have a
// leading zero, so the length plus the value is a unique key and the lookup
// is a constant-time array index with no allocation.
func renderDialCodeIndex(b *bytes.Buffer, terrs []*territory) error {
	byCode := map[string][]*territory{}
	for _, t := range terrs {
		byCode[t.DialCode] = append(byCode[t.DialCode], t)
	}
	codes := make([]string, 0, len(byCode))
	for c := range byCode {
		if l := len(c); l < 1 || l > 3 {
			return fmt.Errorf("calling code %q has unsupported length %d", c, l)
		}
		codes = append(codes, c)
	}
	sort.Strings(codes)

	for _, c := range codes {
		list := byCode[c]
		// The main region for a shared calling code must be tried first.
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].IsMainCountry != list[j].IsMainCountry {
				return list[i].IsMainCountry
			}
			return list[i].Key < list[j].Key
		})
		if len(list) > 1 && !list[0].IsMainCountry {
			return fmt.Errorf("calling code %q is shared by %d regions but none is the main region", c, len(list))
		}
	}

	b.WriteString("// dialCode1, dialCode2 and dialCode3 index metadata by the numeric value of\n")
	b.WriteString("// a calling code of that many digits. The first entry of each slice is the\n")
	b.WriteString("// main region for the code.\n")
	for _, n := range []int{1, 2, 3} {
		size := 1
		for i := 0; i < n; i++ {
			size *= 10
		}
		fmt.Fprintf(b, "var dialCode%d = [%d][]*Metadata{\n", n, size)
		for _, c := range codes {
			if len(c) != n {
				continue
			}
			v, err := strconv.Atoi(c)
			if err != nil {
				return err
			}
			names := make([]string, 0, len(byCode[c]))
			for _, t := range byCode[c] {
				names = append(names, varName(t))
			}
			fmt.Fprintf(b, "\t%d: {%s},\n", v, strings.Join(names, ", "))
		}
		b.WriteString("}\n\n")
	}

	b.WriteString(`// RegionsForCode returns the metadata of every region sharing the calling
// code made of the first n digits of digits, main region first. It returns
// nil when no region uses that code. The lookup performs no allocation.
func RegionsForCode(digits string) []*Metadata {
	switch len(digits) {
	case 1:
		return dialCode1[digits[0]-'0']
	case 2:
		return dialCode2[int(digits[0]-'0')*10+int(digits[1]-'0')]
	case 3:
		return dialCode3[int(digits[0]-'0')*100+int(digits[1]-'0')*10+int(digits[2]-'0')]
	}
	return nil
}

// MaxCallingCodeLen is the length of the longest calling code.
const MaxCallingCodeLen = 3

`)
	return nil
}

func renderTerritory(b *bytes.Buffer, t *territory) {
	fmt.Fprintf(b, "var %s = &Metadata{\n", varName(t))
	fmt.Fprintf(b, "\tISO2: %q,\n\tISO3: %q,\n\tName: %q,\n", t.ISO2, t.ISO3, t.Name)
	fmt.Fprintf(b, "\tDialCode: %q,\n\tIsMainCountry: %t,\n", t.DialCode, t.IsMainCountry)
	if t.LeadingDigits != "" {
		fmt.Fprintf(b, "\tLeadingDigits: %s,\n", lazyPrefix(t.LeadingDigits))
	}
	if t.NationalPrefix != "" {
		fmt.Fprintf(b, "\tNationalPrefix: %q,\n", t.NationalPrefix)
	}
	if t.SimpleNationalPrefix != "" {
		fmt.Fprintf(b, "\tSimpleNationalPrefix: %q,\n", t.SimpleNationalPrefix)
	}
	if t.NationalPrefixForParsing != "" {
		fmt.Fprintf(b, "\tNationalPrefixForParsing: %s,\n", lazyPrefix(t.NationalPrefixForParsing))
	}
	if t.NationalPrefixTransformRule != "" {
		fmt.Fprintf(b, "\tNationalPrefixTransformRule: %q,\n", t.NationalPrefixTransformRule)
	}
	if t.InternationalPrefix != "" {
		fmt.Fprintf(b, "\tInternationalPrefix: %s,\n", lazyFull(t.InternationalPrefix))
	}
	if t.PreferredInternationalPrefix != "" {
		fmt.Fprintf(b, "\tPreferredInternationalPrefix: %q,\n", t.PreferredInternationalPrefix)
	}
	if t.PreferredExtnPrefix != "" {
		fmt.Fprintf(b, "\tPreferredExtnPrefix: %q,\n", t.PreferredExtnPrefix)
	}
	if t.MobileNumberPortable {
		b.WriteString("\tMobileNumberPortable: true,\n")
	}
	fmt.Fprintf(b, "\tMinLength: %d,\n\tMaxLength: %d,\n", t.MinLength, t.MaxLength)

	if !t.General.empty() {
		b.WriteString("\tGeneral: ")
		renderDesc(b, t.General)
		b.WriteString(",\n")
	}
	if !t.NoIntlDialling.empty() {
		b.WriteString("\tNoIntlDialling: ")
		renderDesc(b, t.NoIntlDialling)
		b.WriteString(",\n")
	}

	any := false
	for _, d := range t.Descs {
		if !d.empty() {
			any = true
			break
		}
	}
	if any {
		b.WriteString("\tDescs: [NumDescs]Desc{\n")
		for i, d := range t.Descs {
			if d.empty() {
				continue
			}
			fmt.Fprintf(b, "\t\t%s: ", descNames[i])
			renderDesc(b, d)
			b.WriteString(",\n")
		}
		b.WriteString("\t},\n")
	}

	if len(t.Formats) > 0 {
		b.WriteString("\tFormats: []NumberFormat{\n")
		for _, f := range t.Formats {
			b.WriteString("\t\t{\n")
			fmt.Fprintf(b, "\t\t\tPattern: %s,\n", lazyFull(f.Pattern))
			fmt.Fprintf(b, "\t\t\tFormat: %q,\n", f.Format)
			if f.LeadingDigits != "" {
				fmt.Fprintf(b, "\t\t\tLeadingDigits: %s,\n", lazyPrefix(f.LeadingDigits))
			}
			if f.NationalPrefixRule != "" {
				fmt.Fprintf(b, "\t\t\tNationalPrefixFormattingRule: %q,\n", f.NationalPrefixRule)
			}
			if f.NationalPrefixOptional {
				b.WriteString("\t\t\tNationalPrefixOptional: true,\n")
			}
			if f.CarrierCodeRule != "" {
				fmt.Fprintf(b, "\t\t\tCarrierCodeFormattingRule: %q,\n", f.CarrierCodeRule)
			}
			if f.IntlFormat != "" {
				fmt.Fprintf(b, "\t\t\tIntlFormat: %q,\n", f.IntlFormat)
			}
			b.WriteString("\t\t},\n")
		}
		b.WriteString("\t},\n")
	}

	if len(t.Timezones) > 0 {
		fmt.Fprintf(b, "\tTimezones: []string{%s},\n", quoteJoin(t.Timezones))
	}
	b.WriteString("}\n\n")
}

func renderDesc(b *bytes.Buffer, d desc) {
	b.WriteString("Desc{")
	first := true
	comma := func() {
		if !first {
			b.WriteString(", ")
		}
		first = false
	}
	if d.Pattern != "" {
		comma()
		fmt.Fprintf(b, "Pattern: %s", lazyFull(d.Pattern))
	}
	if len(d.Lengths) > 0 {
		comma()
		fmt.Fprintf(b, "Lengths: []int32{%s}", intsJoin(d.Lengths))
	}
	if len(d.LocalOnly) > 0 {
		comma()
		fmt.Fprintf(b, "LocalOnly: []int32{%s}", intsJoin(d.LocalOnly))
	}
	if d.Example != "" {
		comma()
		fmt.Fprintf(b, "Example: %q", d.Example)
	}
	b.WriteString("}")
}

// lazyFull anchors a pattern to match the whole string.
func lazyFull(p string) string { return fmt.Sprintf("lazyre.Re{Src: %q}", "^(?:"+p+")$") }

// lazyPrefix anchors a pattern to match at the start of the string.
func lazyPrefix(p string) string { return fmt.Sprintf("lazyre.Re{Src: %q}", "^(?:"+p+")") }

func quoteJoin(ss []string) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = strconv.Quote(s)
	}
	return strings.Join(parts, ", ")
}

func intsJoin(ns []int32) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = strconv.Itoa(int(n))
	}
	return strings.Join(parts, ", ")
}
