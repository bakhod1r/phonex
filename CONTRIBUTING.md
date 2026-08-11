# Contributing

Thanks for taking the time. This file covers the two things that are specific
to this repository: what is generated, and what a change has to survive.

## Do not edit generated files

Everything in the following files comes from libphonenumber's data and is
written by `cmd/phonexgen`:

    countries/generated.go
    shortnumber/generated.go
    geocoding/generated.go
    carrier/generated.go
    timezone/generated.go

Change the generator, then run:

    make generate

CI regenerates all five and fails if the result differs from what is
committed, so an edit made by hand will be caught.

## Updating the metadata

The vendored data under `internal/metadata/` is pinned to a libphonenumber
release. To move it:

    make update-metadata METADATA_VERSION=v9.0.33
    make update-prefix-data METADATA_VERSION=v9.0.33
    make generate

The targets extract over the existing tree; review `git status` afterwards to
see what actually moved. Bump `METADATA_VERSION` in the `Makefile` and mention
the new release in `NOTICE` and `CHANGELOG.md` in the same commit.

## What a change has to pass

    make all      # fmt, vet, test, race
    make diff     # differential test against libphonenumber

`make diff` runs in the `difftest/` module, which is separate so that its
comparison dependency never reaches library users. It parses a corpus of
about 12,800 numbers with both implementations and compares the accepted or
rejected verdict, E.164 output, region, type, possibility, formatting,
geocoding, carrier and time zone answers.

A few checks carry a small budget for disagreements, each one traced to
nyaruka bundling data from a different upstream snapshot and documented at
the call site. Run `go test -strict ./...` in `difftest/` to hold every check
to zero. If your change spends budget, say why in the pull request rather
than raising it.

## Behaviour changes

Where libphonenumber and phonex could differ, libphonenumber is right. If you
believe phonex should deviate, open an issue first — a deviation needs a test
in `difftest/` that records it deliberately, not a silently widened budget.

## Tests

New behaviour needs a test that would fail without the change. Tests that
only raise the coverage number without asserting anything are not wanted;
several were removed before the first release.

Fuzz targets live in `fuzz_test.go`. A crash found by fuzzing should land as a
seed under `testdata/fuzz/` together with its fix.

## Commits and pull requests

Commit subjects follow `type: summary` (`feat`, `fix`, `docs`, `test`,
`perf`, `refactor`, `chore`). Explain in the body what the change does and
why, not how.
