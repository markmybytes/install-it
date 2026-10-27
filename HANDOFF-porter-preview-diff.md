# Handoff — Import Preview Diff (manifest hashes)

Status: implemented + verified, **uncommitted** in both repos. Branch: none yet (install-it is on `main`; PR #76 was merged earlier).

## Feature

The Import Preview modal (匯入預覽) now compares the machine's current content against the backup ZIP per import target, using manifest-carried SHA-1 digests. Granularity rule: **one hash per import target** (`ImportOptions`: `settings`, `data`), nothing more.

## Frozen spec

`manifest.json` (format_version stays `1`, `exported_at` unchanged) gains:

```json
"hashes": { "settings": "<sha1 hex lowercase>", "data": "<sha1 hex lowercase>" }
```

Recipe (byte-identical in both repos; golden digest below):

- `settings` = sha1 of the `conf/setting.json` entry/file bytes; last-wins dedupe.
- `data` = fold over entries with arcname `conf/data.db` or under `drivers/` (files only):
  1. dedupe by arcname, **last occurrence wins**
  2. frame = decimal(**UTF-8 byte length** of arcname) + `":"` + arcname + `":"` + per-file-sha1-hex + `"\n"`
  3. sort **full frame strings** lexicographically
  4. sha1 over the concatenation of sorted frames
- Empty key set ⇒ key **omitted** (never `sha1("")`).
- No `contents` hash, no db/drivers split, no mtime/size in hash logic, no `creation_date` field.

**Cross-repo golden** (pinned in Go `pkg/porter/hash_golden_test.go` and Python `tests/test_hashes.py`): fixture `{conf/data.db, drivers/中文/файл.bin, drivers/gpu/amd.bin}` → `data = cb58163e3c641021b923f1268ff49f09eb8082e5`, `settings = 238d439d319aa77425457bf63cbeb3c48c9e91fa`.

## Files changed

### install-it (Go backend + frontend)
- `pkg/porter/manifest.go` — `Hashes struct { Settings, Data string }` added to `Manifest`.
- `pkg/porter/hash.go` (new) — fold recipe, local file/dir hashing, test seam `localFileHashFn`.
- `pkg/porter/action.go` — `toZip`: per-entry sha1 via `io.MultiWriter` during copy, fold after walk, **manifest written last** (order-independence confirmed; aborted exports no longer ship a manifest).
- `pkg/porter/porter.go` — `TargetDiff` value struct (bools + int64 counts/sizes), `ImportPreview.Settings/.Data`; `scanZip` core (ZIP+manifest only) vs `ValidateZip` (core + synchronous live-FS diff); `DownloadAndValidate` sets `step="compare"` + msg before hashing; **imports never re-hash local disk**.
- `pkg/porter/hash_golden_test.go`, `pkg/porter/diff_internal_test.go`, `pkg/porter/porter_test.go` — golden pin, seam tests, settings-only omission test, manifest-last order test.
- `frontend/wailsjs/go/models.ts` — regenerated (`TargetDiff`, `ImportPreview.settings/data`); binding generation clean (no `Not found` warning).
- `frontend/src/pages/porter.vue` — diff state rows on existing Settings/Data checklist lines (`identical`/`changed`/`new`, gray guard reuses `labelNotAvailable`), facts via `formatBytes` + plural `labelDiffFileCount`, `validating` re-entrancy guard on validate/download.
- `frontend/src/components/ProgressModal.vue`, `ProgressStepper.vue` — `compare` step registered between download/complete.
- `frontend/src/i18n/en.json`, `zh_Hant_HK.json` — `labelDiff*` keys + `stepCompare` ("Compare" / "比對").

### it-claws (separate repo)
- `src/it_claws/engine.py` — computes `hashes` from the final `(Path, arcname)` entry list before writing `manifest.json` (which stays last-in-ZIP); streaming 1 MiB chunked `_sha1_file` (kept `usedforsecurity=False`; `hashlib.file_digest` rejected — can't pass the flag).
- `tests/test_hashes.py` (new) — CJK byte-length framing, last-wins dedupe, key omission, rerun stability, integration (manifest last entry).
- `README.md` — "Manifest hashes" section + standalone `python - <<EOF` verifier.

## Decisions log (why not the alternatives)

- **SHA-1** over MD5/SHA-256/CRC32C: purpose is diff signal only; terminal-reproducible requirement (`sha1sum`/`certutil`/`Get-FileHash`) killed CRC32C; perf differences are invisible next to deflate/disk I/O; SHA-256's margin buys nothing here. `usedforsecurity=False` on the Python side.
- **Manifest hashes, not live ZIP hashing**: preview never decompresses (fast-exit). Manifest rides inside the ZIP — its hashes are falsifiable only by re-zipping, which regenerates them. Users editing *live* data without updating anything is exactly what live-hash vs manifest-hash detects.
- **Size/mtime rejected**: SQLite keeps size constant across real content changes; driver mtimes churn on copy; ZIP entry dates unreliable.
- **Synchronous local compare** (user decision) over goroutine + `EventsEmit`; import path kept hash-free so imports never fail on unreadable local files.
- **`data` as one hash** (db + drivers folded): matches the single `Data` import switch; per-part detail stays as facts, not state.
- **No `creation_date` field**: `exported_at` already reports export creation (kept as-is, user confirmed).

## Failure-mode rules

- Legacy backup (no/empty hashes) ⇒ diff rows hidden (`TargetDiff.Available=false`), never faked.
- Row renderable only when incoming hash present **and** agrees with `Has*` presence flags.
- Local file missing ⇒ `LocalPresent=false` → "New" state; local read error ⇒ `Available=false`, never an error.
- DB is hashed live while open (no WAL pragma in `pkg/storage`); torn read ⇒ false "changed", acceptable advisory signal.
- it-claws duplicate arcnames: dedupe lives **inside the hash recipe only** (last-wins); upstream entry list untouched — ZIP bytes unchanged.

## Verification (all green at handoff time)

- `gofmt -l pkg/porter` empty; `go build ./...` ok; `go test ./pkg/... -count=1` all pass.
- `wails generate module` clean.
- Frontend: type-check pass, lint 0 errors (3 pre-existing warnings in `DriverFormComponent.test.ts`), vitest 121/121.
- it-claws: `ruff check` clean, pytest 21 passed.
- Cross-repo fold parity proven (Go fold vs Python `_compute_hashes` on the CJK fixture, byte-identical).

## Deferred / known leftovers (oracle Info-level, deliberate)

- `porter.vue` `diffState` branch `localPresent && !incomingPresent` is unreachable by contract (backend only sets `Available` when incoming present) — harmless guard, left in.
- `Manifest.Hashes` always emitted as `{}` minimum (encoding/json ignores `omitempty` on structs) — documented in `manifest.go`.
- Re-entrancy guard covers validate/download; `ValidateZip` file path itself has no job/cancel (accepted; second click now guarded).
- If backup archives ever cross trust boundaries (shared/hostile sources), revisit SHA-1 + manifest-trust assumptions and consider verify-on-import.

## Next steps

1. Commit both repos (install-it: Go + frontend + bindings; it-claws: engine/tests/README). Suggested: `feat(porter): manifest hashes and import preview diff`.
2. PR against `main`; release notes mention new manifest field (backward compatible both directions).
