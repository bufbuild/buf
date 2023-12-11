# Package stuff

## Notes

- Move `pkg` into own repo
- Move `bufpkg` into own repo
- Mark both `pkg` and `bufpkg` as `DO NOT USE` and never tag
- All buf-specific projects can (and should) use `pkg` and `bufpkg` where it makes sense
- Move `cmd` tools to github.com/bufbuild/tools
- General recommendation is if something lives in `pkg`, use it instead of reinventing the wheel
- The standard for `pkg` packages is pretty high

## Specific Packages

- TODO: Move `protoplugin` out of `app`
- TODO: Look into merging `appcmd` and `appext`
- TODO: Replicate `cobra.PositionalArgs` so that no one needs to directly import cobra
- TODO: Better documentation for app
- TODO: Move x packages to pkg/x, rename to xfilepath, xslices, etc
