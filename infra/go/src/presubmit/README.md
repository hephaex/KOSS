# Presubmit
This directory contains code for fetching new changelists from Gerrit and
testing them against some CI system (currently Jenkins.)

# Building
Build these tools using `go build`.

These tools depend on the `v.io/jiri` package, which can be met by syncing
`https://vanadium.googlesource.com/release.go.jiri`.  If you use `jiri` to sync
Fuchsia, this repo will be checked out for you automatically under
`$WORKDIR/release`, where `$WORKDIR` contains your `.jiri_root`.

We expect these tool to be built and run automatically on the CI system.

# Build example
Given the following directory structure:

```
$WORKDIR/.jiri_root/...
$WORKDIR/infra/go/src/presubmit/...
$WORKDIR/release/go/src/v.io/jiri/...
```

These commands would work, and will generate tools in your current directory.

```
$ export GOPATH=$WORKDIR/infra/go/:$WORKDIR/release/go
$ go build presubmit/query
$ go build presubmit/patch
```

# Running
The presubmit logic is organized into three different tools: `query`, `patch`,
and `report`.

* `query` checks Gerrit for new CLs and sends them to CI for testing.
* `patch` is used by CI to patch the given CLs into its code tree.
* `report` is used by the CI test runner to report its findings to Gerrit.

## query
`query` requires at least one argument: the gerrit host to query.  Running
`query -gerrit https://fuchsia-review.googlesource.com` will look for new CLs
in the Fuchsia repositories and send those refs to a Jenkins instance on
localhost for testing.

The job to which those CLs are sent can be configured with the `-test` argument.
The job must expect parameters in the form of `REFS` and `TESTS`, which are the
CLs to apply and the tests to run, repsectively.

See `query -h` for more options.

## patch
Internally, the CI jobs can use `presubmit/patch` to take the CLs given and
patch its code tree before building and running tests.

See `patch -h` for more options.

## report
Test results are reported to the user via posting comments on the Gerrit CLs.
The authentication code for this process is located in `v.io/jiri/gerrit`, and
the tokens are stored on the CI system via `.gitcookies`.

Most Fuchsia repos also use a concept of `Verified`, which enforces that
presubmit bots have successfully built and tested a change before it can be
submitted.  The rights to mark `Verified` on Gerrit are managed separately from
those required for commenting.  The presubmit bot's account must have both.

When running `report` on your workstation, you may find you don't have rights
to mark changes as `Verified`.

See `report -h` for instructions and options.
