// Package routingpolicy implements multi-tenant webhook routing policies.
//
// It supports YAML/JSON policy schemas with conditions on tenant tier,
// event type, payload size, and time-of-day. Actions include priority
// queue routing, compliance vault routing, rate adjustment, and transform
// application. Policies support hot reload and version history.
package routingpolicy
