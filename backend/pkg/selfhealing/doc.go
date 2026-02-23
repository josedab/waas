// Package selfhealing implements self-healing endpoint discovery.
//
// It supports .well-known/waas-webhooks specs, DNS TXT record lookups,
// HTTP redirect detection, and automatic URL updates with audit trail
// after consecutive delivery failures.
package selfhealing
