/*
Package mocks will have all the mocks of the library, we'll try to use mocking using blackbox
testing and integration tests whenever is possible.
*/
package mocks

// Mutating mocks.
//go:generate mockery -output ./ -dir ../../ -name Runner
