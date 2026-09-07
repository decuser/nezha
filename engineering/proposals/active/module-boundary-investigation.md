> Historical note: this investigation records the now-removed module/import/use/export/load residue from the cleanup sequence. It is retained as cleanup history, not as current runtime state.

# Module Boundary Investigation

## Cut context

Cut 1.2c removes the last registered module-service primitive:

```text
_module_roots
```

`_module_roots` was still registered and cataloged as a service primitive, but it was not exposed by the prelude and had no ordinary user-facing call path. Its implementation only returned `GoRuntime.moduleRegistry.Roots()`. With no default module roots in Nezha, it was inherited service residue rather than language surface.

## Current module-shaped surface

After `_module_roots` is removed, the remaining module-shaped runtime names are:

```text
_import
_use
_export
_load
```

They live in `engine/runtime/hal/substrate/builtins_intrinsic.go` and are registered as evaluator/language intrinsics in `engine/runtime/hal/substrate/register.go`.

Their current behavior is uneven:

```text
_import  validates import call shape, then returns a shaped error: modules are not available
_use     validates use call shape, then returns a shaped error: modules are not available
_export  records exported names on the current environment
_load    reads and evaluates a source file in the current environment
```

So the remaining surface is not a working module system. It is a mixture of:

```text
historical Aiki module syntax/support
active source-file loading
export metadata on environments
stubbed import/use failures
```

## Grammar entry points

`engine/syntax/grammar.ebnfx` still reserves `package` as a keyword and contains `package_stmt`:

```text
package_stmt = "package" STRING
```

`import`, `use`, and `export` are not grammar keywords. They are ordinary calls to privileged names, with raw primitive names gated by authority.

## Prelude exposure

The ordinary prelude still exposes only:

```text
load -> _load
```

It does not expose `import`, `use`, or `export` wrappers.

That makes `load` the only remaining module-adjacent name that is public through the prelude.

## Runtime/module package residue

`engine/runtime/modules` remains present. It contains package scanning, package/help/doc validation, import/use source analysis, FFI-usage analysis, and a registry/cache model.

Current retained Go tests in that package exercise module-source analysis and registry behavior. Those tests preserve historical Aiki module assumptions even though the runtime import/use path now returns "modules are not available".

This package should be treated as an investigation target, not automatically as live Nezha language machinery.

## Boundary reading

For early Nezha, module behavior currently separates into three concerns:

```text
1. package declarations in grammar
2. source loading via load(path)
3. historical module/package registry machinery
```

These should not be conflated.

`load(path)` is not really module import. It is textual/source evaluation in the current environment. It may remain useful as a small-program convenience, but it should not be allowed to smuggle the old module system back into the specimen.

`package`, `_import`, `_use`, `_export`, and `engine/runtime/modules` are stronger candidates for Cut 2 grammar/specimen review.

## Candidate dispositions

```text
_module_roots
  disposition: remove now
  reason: registered service residue; not prelude-exposed; no ordinary call path

load / _load
  disposition: investigate
  reason: public prelude name; source evaluation convenience; not a full module system

package_stmt / package keyword
  disposition: investigate during grammar containment
  reason: grammar-level module residue

_import / _use
  disposition: likely remove or keep as intentional negative stubs until grammar decision
  reason: currently validate then report modules unavailable

_export
  disposition: investigate with package/import/use
  reason: records environment exports even though the module system is not active

engine/runtime/modules
  disposition: parked/investigate
  reason: historical Aiki module machinery; may be useful evidence, but is not currently live ordinary module loading
```

## Next questions

```text
Does Nezha need package declarations before eval-spec work?
Does Nezha need load(path), or can examples/smoke stay single-file?
Should import/use/export be removed before grammar containment, or kept as visible stubs to guide the cut?
Should engine/runtime/modules be parked, deleted, or moved under engineering/archival evidence?
```

## Gate for this cut

```sh
make validate-lang
```
