# Release Process

The guide describes how to release a new version of the kcp-operator.

## Prerequisites

1. Have all desired changes merged and/or cherrypicked into the appropriate
   release branch.

## Minor Release

Minor releases (0.x) are tagged directly on the `main` branch and the `v0.X.0`
tag represents where the corresponding `release/v0.X` branch branches off.

1. Checkout the desired `main` branch commit.
1. Tag the main module: `git tag -m "version 0.X" v0.X.0`
1. Tag the SDK module: `git tag -m "SDK version 0.X" sdk/v0.X.0`
1. Push the tags: `git push upstream v0.X.0 sdk/v0.X.0`
1. Create the release branch: `git checkout -B release/v0.X`
1. Push the release branch: `git push -u upstream release/v0.X`

## Patch Releases

Patch releases (v0.x.y) are tagged with in a release branch.

1. Checkout the desired `release/v0.X` branch commit.
1. Tag the main module: `git tag -m "version 0.X.Y" v0.X.Y`
1. Tag the SDK module: `git tag -m "SDK version 0.X.Y" sdk/v0.X.Y`
1. Push the tags: `git push upstream v0.X.Y sdk/v0.X.Y`
