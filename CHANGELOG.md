# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[semantic versioning](https://semver.org/spec/v2.0.0.html).

While the major version is 0, the API may change in a minor release. Each such
change will be listed here.

## [Unreleased]

## [0.2.0] - 2026-08-11

### Added

- `GenerateWith(region, type, intn)` — number generation with the randomness
  supplied by the caller, so that output can be reproduced from a seed.
  `Generate` and `GenerateForType` now call it with the global `math/rand`.
- `AnyType`, asking for whatever range a region defines rather than a
  particular one.
- `GenerateForPrefix(region, prefix, intn)` — a valid number whose national
  number starts with a given area or operator code. The prefix may be written
  with or without the trunk digit; the shape that fits it is cached.

### Fixed

- `Generate` and `GenerateForType` returned the same example number on every
  call for countries that do not separate their fixed-line and mobile ranges
  (the United States and Canada among them). Such a region reports every
  number as `FIXED_LINE_OR_MOBILE`, which the type check rejected, so no
  randomised candidate was ever accepted.

## [0.1.0] - 2026-08-11

First release.

### Added

- **Parsing** of international, national and RFC 3966 forms, with the
  region's own trunk prefix and IDD rules, extensions, and optional vanity
  letters. Parsing into an existing `Phone` performs no allocations.
- **Validation**: `IsValid` against the region's assigned ranges,
  `IsPossible` for length alone, and `Possibility` for the reason a length
  check failed.
- **Number types**: the eleven libphonenumber ranges, with a predicate each.
- **Formatting**: E.164, international, national, RFC 3966 and
  `OutOfCountry`, plus `AppendE164` for an allocation-free write.
- **`Formatter`**, an as-you-type formatter for input fields.
- **`Match`**, grading two numbers as exact, NSN, short-NSN or no match.
- **Country metadata**: lookup by region, calling code or number, search,
  and libphonenumber's example numbers.
- **Encoding**: `json`, `encoding.TextMarshaler`, `sql.Scanner` and
  `driver.Valuer`, all round-tripping through E.164.
- **Privacy helpers**: masking, and SHA-256 or HMAC fingerprints over the
  canonical form.
- **`shortnumber`**, a subpackage for short numbers such as 112 and 911,
  including `ConnectsToEmergency`.
- **`geocoding`**, **`carrier`** and **`timezone`** subpackages, each with its
  own data, kept separate so that a program which does not need them does not
  link them in.
- **`cmd/phonexgen`**, which generates every table from the libphonenumber
  data vendored under `internal/metadata/`.
- **`difftest/`**, a separate module comparing this library against
  libphonenumber over 12785 ordinary numbers and 8044 short numbers.
- **Carrier selection codes**, kept off the national number and reported by
  `CarrierCode`, with `NationalWithCarrier` to put one back for display.

### Metadata

Generated from libphonenumber **v9.0.32**.
