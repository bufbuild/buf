# bufls

`bufls` is a Protobuf language server compatible with Buf modules
and workspaces.

## Usage

Regradless of the LSP-compatible editor you use, you'll need to install
`bufls` so that it's available on your `PATH`.

### Vim

With [vim-lsp], the only configuration you need is the following:

```vim
Plug 'prabirshrestha/vim-lsp'

augroup LspBuf
  au!
  autocmd User lsp_setup call lsp#register_server({
      \ 'name': 'bufls',
      \ 'cmd': {server_info->['bufls', 'serve']},
      \ 'whitelist': ['proto'],
      \ })
  autocmd FileType proto nmap <buffer> gd <plug>(lsp-definition)
augroup END
```

  [vim-lsp]: https://github.com/prabirshrestha/vim-lsp

## Supported features

Buf's language server behaves similarly to the rest of the `buf` CLI. If
the user has a `buf.work.yaml` defined, the modules defined in the workspace
will take precedence over the modules specified in the `buf.lock` (i.e. the
modules found in the module cache). The language server requires that inputs
are of the [protofile] type.

  [protofile]: https://docs.buf.build/reference/inputs#protofile

### Go to definition

Go to definition resolves the definition location of a symbol at a
given text document position (i.e. [textDocument/definition]).

This feature is currently only implemented on the `textDocument/definition`
endpoint. It may make sense to move this to `textDocument/typeDefinition`
and/or `textDocument/typeImplementation`. The Protobuf grammar is far more
limited than a programming language grammar, so not all of the semantics
for each LSP endpoint apply here.

Today, this feature is only supported for messages and enums. The well-known
types (WKT), and [custom] options are not yet supported.

  [textDocument/definition]: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#textDocument_definition

## Implementation

Protobuf compilation is _fast_, so the implementation is currently naive. Every
editor command will compile the input file (e.g. `file://proto/pet/v1/pet.proto`)
from scratch (there isn't any caching). Simple caching is fairly straightforward,
but the cache would need to be cleared whenever a file is edited during the same
language server session, which would require a file watcher. For now, performance
is fine as-is (even for workspaces and large modules), but we might need to revisit
this later as build graphs continue to grow.

## Future work

### TODO

 - Add go to definition support for WKT (i.e. synthesize the WKT in the module cache).
 - Add go to definition support for [custom] options.

### Tree-sitter

A [tree-sitter] grammar would be useful for a subset of LSP features that need access to
a single `.proto` file (e.g. syntax highlighting, folding, etc), or for code navigation
and any other features that benefit from tagging and captures (e.g. go to definition).

Unfortunately, tree-sitter doesn't have a stable version of Go bindings listed in their
[supported set]. There is an in-progress implementation at [smacker/go-tree-sitter], but
it is incomplete and (unsurprisingly) requires a dependency on Cgo.

With that said, [tree-sitter] is a performance optimization - it has no effect on the language
server's functionality. In other words, it should be thought of as an implementation detail
that we can consider later (if ever).

  [tree-sitter]: https://tree-sitter.github.io/tree-sitter
  [supported set]: https://tree-sitter.github.io/tree-sitter/#language-bindings
  [smacker/go-tree-sitter]: https://github.com/smacker/go-tree-sitter
