# LSP Incremental Document Sync Implementation

## Overview

This implementation provides robust support for LSP incremental document synchronization by properly distinguishing between full document sync and incremental edits.

## The Problem

The `go.lsp.dev/protocol` library uses a non-pointer `Range` field in `TextDocumentContentChangeEvent`, making it impossible to distinguish between:
- **Full document sync**: Range field omitted in JSON → unmarshals to zero-valued Range
- **Incremental edit at 0,0**: Range explicitly set to `{start: {0,0}, end: {0,0}}` → also zero-valued Range

Without proper detection, inserting text at the start of a document (position 0,0) would be incorrectly treated as a full document replacement, corrupting the file.

## The Solution

### 1. Custom Range Detection (`file.go`)

**`detectRangePresence()`**: Analyzes raw JSON to determine if the `range` field was actually present:
```go
func detectRangePresence(rawChanges []json.RawMessage) ([]bool, error)
```

### 2. Dual ApplyChanges Methods (`file.go`)

**`ApplyChanges()`**: Maintains backward compatibility using a heuristic (Range is zero AND text > 100 chars = likely full sync)

**`ApplyChangesWithRangeInfo()`**: Uses explicit range presence information for accurate handling

### 3. Request Interception (`buflsp.go`)

**`handleDidChangeWithRangeDetection()`**: Intercepts `textDocument/didChange` requests before normal processing:
1. Extracts raw JSON params
2. Detects which changes have Range present
3. Calls `ApplyChangesWithRangeInfo()` with accurate metadata

### 4. Server Type Change (`server.go`)

Changed `newServer()` return type from `protocol.Server` (interface) to `*server` (concrete type) to allow direct method calls during request interception.

## How It Works

```
Client sends textDocument/didChange
         ↓
AsyncHandler in buflsp.go intercepts
         ↓
handleDidChangeWithRangeDetection() analyzes raw JSON
         ↓
detectRangePresence() checks for "range" key
         ↓
ApplyChangesWithRangeInfo() applies changes correctly
         ↓
Full sync: Replace entire document
Incremental: Apply range-based edit
```

## Testing

All existing tests pass, including:
- `TestIncrementalDocumentSync`: Verifies incremental edits work correctly
- `TestIncrementalDocumentSyncMultipleChanges`: Tests sequential changes
- `TestIncrementalDocumentSyncReplace`: Tests text replacement
- `TestFullDocumentSyncStillWorks`: Ensures backward compatibility

## What LSP Clients Get

✅ **Proper incremental sync**: Clients can now send true incremental updates without risk of corruption

✅ **Efficient editing**: Only changed portions are transmitted, not the entire document

✅ **Zero-position edits**: Inserting text at document start (0,0) now works correctly

✅ **Backward compatibility**: Full document sync still works when Range is omitted

## Files Modified

- `private/buf/buflsp/file.go`: Added range detection and dual ApplyChanges methods
- `private/buf/buflsp/buflsp.go`: Added request interception for didChange
- `private/buf/buflsp/server.go`: Changed newServer return type to concrete *server
- `private/buf/buflsp/document_sync_test.go`: Comprehensive tests (already existed)
