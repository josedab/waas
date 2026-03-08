// Package main implements the delivery engine binary for WaaS. It consumes
// webhook delivery jobs from a queue, dispatches HTTP requests to configured
// endpoints, and manages retry logic with exponential backoff.
package main
