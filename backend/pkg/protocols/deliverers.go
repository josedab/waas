// Package protocols implements webhook delivery over multiple protocols.
// The deliverer implementations are organized by protocol:
//   - http_deliverer.go: HTTP/HTTPS webhook delivery
//   - grpc_deliverer.go: gRPC webhook delivery
//   - ws_deliverer.go: WebSocket webhook delivery
//   - mqtt_deliverer.go: MQTT webhook delivery
//   - kafka_deliverer.go: Apache Kafka delivery
//   - sns_sqs_deliverer.go: AWS SNS and SQS delivery
//   - protocol_adapters.go: Unified protocol adapter layer with observability
package protocols
