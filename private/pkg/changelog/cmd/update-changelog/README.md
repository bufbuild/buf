# update-changelog

This CLI tool is designed to update a changelog file in a specific format. It provides two operations: "release" and "unrelease". The tool is intended to be used with a changelog file in the Markdown format.


## Usage
The CLI tool has the following commands:

### Release
The "release" command is used to release the changelog in a new version. It requires the following arguments:

```bash
update-changelog release --version <version>
```

CHANGELOG.md:
```diff
--- changelog.1.md      2023-06-23 15:04:56
+++ changelog.2.md      2023-06-23 15:05:19
@@ -1,10 +1,10 @@
 # Changelog
 
-## [Unreleased]
+## [v1.0.1] - 2020-01-02
 
 - Change foobar
 
 ## [v1.0.0] - 2020-01-01
 
-[Unreleased]: https://github.com/foobar/foo/compare/v1.0.0...HEAD
+[v1.0.1]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.1
 [v1.0.0]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.0
```

### Unrelease
The "unrelease" command is used add the `Unreleased` section back into a changelog after a release has been made. It requires no arguments:

```bash
updatechangelog unrelease
```

CHANGELOG.md:
```diff
--- changelog.1.md      2023-06-23 15:09:23
+++ changelog.2.md      2023-06-23 15:09:34
@@ -1,10 +1,15 @@
 # Changelog
 
+## [Unreleased]
+
+- No changes yet.
+
 ## [v1.0.1] - 2020-01-02
 
 - Change foobar
 
 ## [v1.0.0] - 2020-01-01
 
+[Unreleased]: https://github.com/foobar/foo/compare/v1.0.1...HEAD
 [v1.0.1]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.1
 [v1.0.0]: https://github.com/foobar/foo/compare/v1.0.0...v1.0.0
```



