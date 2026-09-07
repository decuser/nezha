// Package aitest owns the Nezha test helper surface used by the `nezha test` command.
package aitest

// Source is prepended by the test command. It is not a shipped Nezha module.
const Source = `let equal = (actual, expected) {
	_test_equal(actual, expected)
}

let not_equal = (actual, expected) {
	_test_not_equal(actual, expected)
}

let is_true = (val) {
	_test_true(val)
}

let is_false = (val) {
	_test_false(val)
}

let error = (val) {
	_test_error(val)
}

let not_error = (val) {
	_test_not_error(val)
}

let run = (name, fn) {
	_test_run(name, fn)
}

let faults = (fn) {
	_test_faults(fn)
}
`
