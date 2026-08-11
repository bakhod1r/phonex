# Security policy

## Supported versions

The most recent minor release is supported. While the major version is 0,
fixes go into a new patch release of that line rather than being backported.

## Reporting a vulnerability

Report privately through GitHub's
[security advisory form](https://github.com/bakhod1r/phonex/security/advisories/new).
Please do not open a public issue for a vulnerability.

Include the input that triggers it, the version, and what happens. You should
get an acknowledgement within a week.

## Scope

phonex parses untrusted text, so the interesting cases are inputs that make it
misbehave rather than merely return the wrong answer:

- a panic, an out-of-range index, or unbounded memory growth on any input
- input that makes parsing take time disproportionate to its length
- a number that validates as one region or type but formats as another in a
  way that could be used to disguise it

A wrong answer that matches libphonenumber is a metadata question for
upstream, not a vulnerability here. Report it as an ordinary issue.

Parsing is fuzzed in CI, and `Parse` writes into fixed-size buffers, so
oversized input is rejected rather than allocated for.
