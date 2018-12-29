/*
Package mocks will have all the mocks of the library, we'll try to use mocking using blackbox
testing and integration tests whenever is possible.
*/
package mocks

// Runner mocks.
//go:generate mockery -output ./ -dir ../../ -name Runner

// concurrencylimit mocks.
//go:generate mockery -output ./concurrencylimit/execute -dir ../../concurrencylimit/execute -name Executor
//go:generate mockery -output ./concurrencylimit/limit -dir ../../concurrencylimit/limit -name Limiter
