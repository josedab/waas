/**
 * Flow Builder - Types
 * Maps to backend pkg/flowbuilder models
 */

export type NodeType =
  | 'trigger'
  | 'transform'
  | 'filter'
  | 'split'
  | 'join'
  | 'delay'
  | 'http_call'
  | 'condition'
  | 'switch'
  | 'loop'
  | 'aggregate'
  | 'notify'
  | 'end';

export type WorkflowStatus = 'draft' | 'active' | 'paused' | 'archived';
export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'timed_out';

export interface Position {
  x: number;
  y: number;
}

export interface WorkflowNode {
  id: string;
  workflowId: string;
  type: NodeType;
  name: string;
  config: Record<string, unknown>;
  position: Position;
  timeoutSeconds: number;
  retryCount: number;
}

export interface WorkflowEdge {
  id: string;
  workflowId: string;
  sourceNodeId: string;
  targetNodeId: string;
  condition?: string;
  label?: string;
  priority: number;
}

export interface Workflow {
  id: string;
  tenantId: string;
  name: string;
  description: string;
  status: WorkflowStatus;
  version: number;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  variables?: Record<string, unknown>;
  maxTimeoutSeconds: number;
  maxRetries: number;
  totalExecutions: number;
  successRate: number;
  createdAt: string;
  updatedAt: string;
}

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  icon: string;
}

export interface ExecutionVisualization {
  nodeId: string;
  status: ExecutionStatus;
  startedAt?: string;
  completedAt?: string;
  output?: unknown;
  error?: string;
  durationMs?: number;
}

/** Node type metadata for the palette */
export interface NodeTypeInfo {
  type: NodeType;
  label: string;
  description: string;
  icon: string;
  color: string;
  category: 'trigger' | 'logic' | 'action' | 'flow';
  maxInputs: number;
  maxOutputs: number;
}

export const NODE_TYPE_INFO: NodeTypeInfo[] = [
  { type: 'trigger', label: 'Trigger', description: 'Start the workflow on an event', icon: '⚡', color: '#3b82f6', category: 'trigger', maxInputs: 0, maxOutputs: 1 },
  { type: 'transform', label: 'Transform', description: 'Transform the payload', icon: '🔄', color: '#8b5cf6', category: 'action', maxInputs: 1, maxOutputs: 1 },
  { type: 'filter', label: 'Filter', description: 'Filter events by condition', icon: '🔍', color: '#f59e0b', category: 'logic', maxInputs: 1, maxOutputs: 2 },
  { type: 'condition', label: 'Condition', description: 'Branch based on condition', icon: '🔀', color: '#ef4444', category: 'logic', maxInputs: 1, maxOutputs: 2 },
  { type: 'http_call', label: 'HTTP Request', description: 'Make an HTTP request', icon: '🌐', color: '#10b981', category: 'action', maxInputs: 1, maxOutputs: 1 },
  { type: 'delay', label: 'Delay', description: 'Wait before continuing', icon: '⏱️', color: '#6366f1', category: 'flow', maxInputs: 1, maxOutputs: 1 },
  { type: 'split', label: 'Fan Out', description: 'Send to multiple targets', icon: '📤', color: '#ec4899', category: 'flow', maxInputs: 1, maxOutputs: 5 },
  { type: 'join', label: 'Join', description: 'Wait for multiple inputs', icon: '📥', color: '#14b8a6', category: 'flow', maxInputs: 5, maxOutputs: 1 },
  { type: 'switch', label: 'Switch', description: 'Route to different paths', icon: '🔃', color: '#f97316', category: 'logic', maxInputs: 1, maxOutputs: 5 },
  { type: 'loop', label: 'Loop', description: 'Iterate over items', icon: '🔁', color: '#a855f7', category: 'flow', maxInputs: 1, maxOutputs: 1 },
  { type: 'aggregate', label: 'Aggregate', description: 'Collect and combine data', icon: '📊', color: '#06b6d4', category: 'action', maxInputs: 1, maxOutputs: 1 },
  { type: 'notify', label: 'Notify', description: 'Send notification', icon: '🔔', color: '#eab308', category: 'action', maxInputs: 1, maxOutputs: 1 },
  { type: 'end', label: 'End', description: 'End the workflow', icon: '🏁', color: '#6b7280', category: 'flow', maxInputs: 1, maxOutputs: 0 },
];

export const WORKFLOW_TEMPLATES: WorkflowTemplate[] = [
  {
    id: 'basic-webhook-forward',
    name: 'Basic Webhook Forward',
    description: 'Receive a webhook and forward it to an HTTP endpoint',
    category: 'starter',
    icon: '🚀',
    nodes: [
      { id: 'trigger-1', workflowId: '', type: 'trigger', name: 'Webhook Received', config: { eventType: 'webhook.received' }, position: { x: 100, y: 200 }, timeoutSeconds: 30, retryCount: 0 },
      { id: 'http-1', workflowId: '', type: 'http_call', name: 'Forward Webhook', config: { method: 'POST', url: '' }, position: { x: 400, y: 200 }, timeoutSeconds: 30, retryCount: 3 },
      { id: 'end-1', workflowId: '', type: 'end', name: 'Done', config: {}, position: { x: 700, y: 200 }, timeoutSeconds: 0, retryCount: 0 },
    ],
    edges: [
      { id: 'e1', workflowId: '', sourceNodeId: 'trigger-1', targetNodeId: 'http-1', priority: 1 },
      { id: 'e2', workflowId: '', sourceNodeId: 'http-1', targetNodeId: 'end-1', priority: 1 },
    ],
  },
  {
    id: 'filter-and-transform',
    name: 'Filter & Transform',
    description: 'Filter events, transform payload, then deliver',
    category: 'starter',
    icon: '🔄',
    nodes: [
      { id: 'trigger-1', workflowId: '', type: 'trigger', name: 'Event Received', config: {}, position: { x: 100, y: 200 }, timeoutSeconds: 30, retryCount: 0 },
      { id: 'filter-1', workflowId: '', type: 'filter', name: 'Filter Events', config: { expression: 'event.type == "order.created"' }, position: { x: 300, y: 200 }, timeoutSeconds: 10, retryCount: 0 },
      { id: 'transform-1', workflowId: '', type: 'transform', name: 'Transform Payload', config: { template: '{}' }, position: { x: 500, y: 200 }, timeoutSeconds: 10, retryCount: 0 },
      { id: 'http-1', workflowId: '', type: 'http_call', name: 'Deliver', config: { method: 'POST' }, position: { x: 700, y: 200 }, timeoutSeconds: 30, retryCount: 3 },
    ],
    edges: [
      { id: 'e1', workflowId: '', sourceNodeId: 'trigger-1', targetNodeId: 'filter-1', priority: 1 },
      { id: 'e2', workflowId: '', sourceNodeId: 'filter-1', targetNodeId: 'transform-1', label: 'pass', priority: 1 },
      { id: 'e3', workflowId: '', sourceNodeId: 'transform-1', targetNodeId: 'http-1', priority: 1 },
    ],
  },
  {
    id: 'fan-out-delivery',
    name: 'Fan-Out Delivery',
    description: 'Receive event and deliver to multiple endpoints in parallel',
    category: 'advanced',
    icon: '📤',
    nodes: [
      { id: 'trigger-1', workflowId: '', type: 'trigger', name: 'Event', config: {}, position: { x: 100, y: 250 }, timeoutSeconds: 30, retryCount: 0 },
      { id: 'split-1', workflowId: '', type: 'split', name: 'Fan Out', config: {}, position: { x: 350, y: 250 }, timeoutSeconds: 10, retryCount: 0 },
      { id: 'http-1', workflowId: '', type: 'http_call', name: 'Target A', config: { method: 'POST' }, position: { x: 600, y: 100 }, timeoutSeconds: 30, retryCount: 3 },
      { id: 'http-2', workflowId: '', type: 'http_call', name: 'Target B', config: { method: 'POST' }, position: { x: 600, y: 400 }, timeoutSeconds: 30, retryCount: 3 },
    ],
    edges: [
      { id: 'e1', workflowId: '', sourceNodeId: 'trigger-1', targetNodeId: 'split-1', priority: 1 },
      { id: 'e2', workflowId: '', sourceNodeId: 'split-1', targetNodeId: 'http-1', priority: 1 },
      { id: 'e3', workflowId: '', sourceNodeId: 'split-1', targetNodeId: 'http-2', priority: 2 },
    ],
  },
];
