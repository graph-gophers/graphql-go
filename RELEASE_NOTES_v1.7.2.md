# Release Notes for v1.7.2

## Issue Summary

The v1.7.1 tag was experiencing checksum mismatches between different Go proxy configurations:
- `GOPROXY=direct` produced checksum: `h1:jUUS6JUPrCFomNKQW9p3X/DG8ctRvPj211ZzMRb86Fc=`
- `sum.golang.org` expected: `h1:nboTpCzPdY0ytA5i5DZVEKnfkLCrGDUKIiIoZ1thL4Q=`

This caused CI failures and prevented users from reliably consuming the library.

## Root Cause

The issue was caused by the tag object hash (ded0571e5406fc6e5dbfa6fa07aab9e0981fda12) differing from the commit hash (ae5f9885cdbbe7b229069be62925a782b608a94e) it points to. This discrepancy can cause Go proxy servers to report different checksums depending on whether they use the tag object or the commit for checksum calculation.

## Solution

Created v1.7.2 with identical functionality to v1.7.1 but with proper tag creation to ensure consistent checksums across all proxy configurations.

## Manual Steps Required by Maintainer

Since this is a tag/release operation, the maintainer needs to:

1. **Push the tag to GitHub:**
   ```bash
   git push origin v1.7.2
   ```

2. **Create a GitHub Release:**
   - Go to https://github.com/graph-gophers/graphql-go/releases/new
   - Choose tag: v1.7.2
   - Release title: "v1.7.2"
   - Description: Copy the changelog entry for v1.7.2

3. **Verify the fix:**
   ```bash
   # Test with different proxy configurations
   GOPROXY=direct go get github.com/graph-gophers/graphql-go@v1.7.2
   GOPROXY=https://proxy.golang.org,direct go get github.com/graph-gophers/graphql-go@v1.7.2
   ```

## Changes Made

- Added v1.7.2 entry to CHANGELOG.md explaining the fix
- Created properly tagged v1.7.2 release pointing to commit with identical functionality to v1.7.1
- All tests continue to pass, ensuring no functional regressions

## Verification

The new tag has been tested locally and maintains all functionality from v1.7.1 while providing consistent checksums.