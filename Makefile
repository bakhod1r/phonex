.PHONY: all test bench vet fmt cover generate update-metadata update-prefix-data diff fuzz

# The metadata is pinned to a tagged libphonenumber release, not to master,
# so that a regeneration is a deliberate, reviewable step.
METADATA_VERSION := v9.0.32
METADATA_BASE    := https://raw.githubusercontent.com/google/libphonenumber/$(METADATA_VERSION)/resources
TARBALL_URL      := https://github.com/google/libphonenumber/archive/refs/tags/$(METADATA_VERSION).tar.gz
METADATA         := internal/metadata/PhoneNumberMetadata.xml
SHORT_METADATA   := internal/metadata/ShortNumberMetadata.xml
PREFIX_DATA      := internal/metadata/timezones internal/metadata/carrier internal/metadata/geocoding

all: fmt vet test

test:
	go test ./...
	go test -race ./...

# Compare against Google's libphonenumber, through nyaruka/phonenumbers.
diff:
	cd difftest && go test -v ./...

fuzz:
	go test -run '^$$' -fuzz FuzzParse -fuzztime 60s .

bench:
	go test -run '^$$' -bench . -benchmem ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# Regenerate countries/generated.go from the vendored upstream XML.
generate:
	go generate ./...

# Pull the pinned release of libphonenumber's metadata and regenerate. Review
# the diff before committing: upstream changes number ranges, not just
# formats. Bump METADATA_VERSION to move to a newer release.
update-metadata:
	curl -fsSL $(METADATA_BASE)/PhoneNumberMetadata.xml -o $(METADATA)
	curl -fsSL $(METADATA_BASE)/ShortNumberMetadata.xml -o $(SHORT_METADATA)
	$(MAKE) update-prefix-data
	$(MAKE) generate
	$(MAKE) test
	$(MAKE) diff

# The geocoding, carrier and time zone data are hundreds of files, so they
# come from the release tarball rather than one request each. Only the English
# geocoding and carrier data is vendored; adding a language means adding its
# directory here and a package to generate from it.
#
# The extraction writes over the existing tree rather than replacing it, so
# after running this check "git status" for files upstream has removed and
# delete them yourself. A stale file would otherwise keep contributing
# prefixes to the generated tables.
update-prefix-data:
	@tmp=$$(mktemp -d) && \
	curl -fsSL $(TARBALL_URL) -o $$tmp/lpn.tar.gz && \
	tar xzf $$tmp/lpn.tar.gz --strip-components=2 -C internal/metadata \
		libphonenumber-$(METADATA_VERSION:v%=%)/resources/timezones \
		libphonenumber-$(METADATA_VERSION:v%=%)/resources/carrier/en \
		libphonenumber-$(METADATA_VERSION:v%=%)/resources/geocoding/en && \
	echo "refreshed $(PREFIX_DATA) — review git status for files upstream removed"
