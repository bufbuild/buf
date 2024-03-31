# Imports cache

This cache is used for the imports tests. It's caching an exact copy of the `school`, people`, and `students`
modules in the `success` dir.

To understand how this cache is built, see the `bufmodulecache` pkg.

To re-generate digests:

```
buf-digest \
  private/buf/cmd/buf/testdata/imports/success/school \
  private/buf/cmd/buf/testdata/imports/success/people \
  private/buf/cmd/buf/testdata/imports/success/students
```

To make new commit IDs:

```
buf-new-commit-id
```
