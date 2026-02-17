// Package protocols implements webhook delivery over multiple protocols.
// The deliverer implementations are organized by protocol:
//   - http_deliverer.go: HTTP/HTTPS webhook delivery
//   - grpc_deliverer.go: gRPC webhook delivery
//   - ws_deliverer.go: WebSocket webhook delivery
//   - mqtt_deliverer.go: MQTT webhook delivery
package protocols
