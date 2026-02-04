/**
 * FlowBuilder - Visual DAG workflow editor
 * 
 * Renders an interactive canvas for composing webhook pipelines:
 * trigger → transform → filter → fan-out → deliver
 * 
 * Uses SVG for edge rendering and absolute positioning for nodes.
 * For production, integrate with @xyflow/react (React Flow) for full
 * drag-and-drop, zooming, and minimap support.
 */

import React, { useState, useCallback, useRef } from 'react';
import {
  Workflow,
  WorkflowNode,
  WorkflowEdge,
  NodeType,
  NODE_TYPE_INFO,
  WORKFLOW_TEMPLATES,
  WorkflowTemplate,
  ExecutionVisualization,
  Position,
} from './types';

// ----- Node Component -----

interface FlowNodeProps {
  node: WorkflowNode;
  selected: boolean;
  execution?: ExecutionVisualization;
  onSelect: (id: string) => void;
  onDragStart: (id: string, e: React.MouseEvent) => void;
}

const FlowNode: React.FC<FlowNodeProps> = ({ node, selected, execution, onSelect, onDragStart }) => {
  const info = NODE_TYPE_INFO.find((n) => n.type === node.type);
  const statusColor = execution
    ? execution.status === 'completed' ? '#10b981'
    : execution.status === 'running' ? '#3b82f6'
    : execution.status === 'failed' ? '#ef4444'
    : '#6b7280'
    : undefined;

  return (
    <div
      style={{
        position: 'absolute',
        left: node.position.x,
        top: node.position.y,
        width: 160,
        padding: '12px',
        background: selected ? '#1e293b' : '#0f172a',
        border: `2px solid ${statusColor || (selected ? '#3b82f6' : info?.color || '#334155')}`,
        borderRadius: '8px',
        cursor: 'grab',
        userSelect: 'none',
        color: '#e2e8f0',
        fontSize: '13px',
        boxShadow: selected ? '0 0 0 2px rgba(59,130,246,0.3)' : undefined,
      }}
      onClick={() => onSelect(node.id)}
      onMouseDown={(e) => onDragStart(node.id, e)}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '4px' }}>
        <span>{info?.icon || '📦'}</span>
        <strong style={{ fontSize: '12px' }}>{node.name}</strong>
      </div>
      <div style={{ color: '#64748b', fontSize: '11px' }}>{info?.label}</div>
      {execution && (
        <div style={{ marginTop: '6px', fontSize: '10px', color: statusColor }}>
          {execution.status} {execution.durationMs ? `(${execution.durationMs}ms)` : ''}
        </div>
      )}
    </div>
  );
};

// ----- Edge SVG Component -----

interface FlowEdgeProps {
  edge: WorkflowEdge;
  sourcePos: Position;
  targetPos: Position;
}

const FlowEdge: React.FC<FlowEdgeProps> = ({ edge, sourcePos, targetPos }) => {
  const sx = sourcePos.x + 160;
  const sy = sourcePos.y + 30;
  const tx = targetPos.x;
  const ty = targetPos.y + 30;
  const cx = (sx + tx) / 2;

  return (
    <g>
      <path
        d={`M ${sx} ${sy} C ${cx} ${sy}, ${cx} ${ty}, ${tx} ${ty}`}
        fill="none"
        stroke="#475569"
        strokeWidth={2}
      />
      {edge.label && (
        <text x={cx} y={(sy + ty) / 2 - 8} fill="#94a3b8" fontSize="11" textAnchor="middle">
          {edge.label}
        </text>
      )}
      <circle cx={tx} cy={ty} r={3} fill="#475569" />
    </g>
  );
};

// ----- Node Palette -----

interface NodePaletteProps {
  onAddNode: (type: NodeType) => void;
}

const NodePalette: React.FC<NodePaletteProps> = ({ onAddNode }) => (
  <div style={{ width: 200, background: '#1e293b', padding: '12px', borderRight: '1px solid #334155', overflowY: 'auto' }}>
    <h3 style={{ color: '#e2e8f0', fontSize: '14px', marginBottom: '12px' }}>Node Palette</h3>
    {['trigger', 'logic', 'action', 'flow'].map((cat) => (
      <div key={cat} style={{ marginBottom: '12px' }}>
        <div style={{ color: '#94a3b8', fontSize: '11px', textTransform: 'uppercase', marginBottom: '6px' }}>{cat}</div>
        {NODE_TYPE_INFO.filter((n) => n.category === cat).map((info) => (
          <button
            key={info.type}
            onClick={() => onAddNode(info.type)}
            style={{
              display: 'flex', alignItems: 'center', gap: '6px', width: '100%', padding: '6px 8px',
              background: 'transparent', border: '1px solid #334155', borderRadius: '6px',
              color: '#e2e8f0', fontSize: '12px', cursor: 'pointer', marginBottom: '4px', textAlign: 'left',
            }}
          >
            <span>{info.icon}</span> {info.label}
          </button>
        ))}
      </div>
    ))}
  </div>
);

// ----- Template Selector -----

interface TemplateSelectorProps {
  onSelect: (template: WorkflowTemplate) => void;
  onClose: () => void;
}

const TemplateSelector: React.FC<TemplateSelectorProps> = ({ onSelect, onClose }) => (
  <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }}>
    <div style={{ background: '#1e293b', borderRadius: '12px', padding: '24px', maxWidth: 600, width: '90%' }}>
      <h2 style={{ color: '#e2e8f0', marginBottom: '16px' }}>Choose a Template</h2>
      <div style={{ display: 'grid', gap: '12px' }}>
        {WORKFLOW_TEMPLATES.map((t) => (
          <button
            key={t.id}
            onClick={() => { onSelect(t); onClose(); }}
            style={{
              background: '#0f172a', border: '1px solid #334155', borderRadius: '8px',
              padding: '12px', cursor: 'pointer', textAlign: 'left', color: '#e2e8f0',
            }}
          >
            <div style={{ fontSize: '16px', marginBottom: '4px' }}>{t.icon} {t.name}</div>
            <div style={{ color: '#94a3b8', fontSize: '12px' }}>{t.description}</div>
          </button>
        ))}
      </div>
      <button onClick={onClose} style={{ marginTop: '12px', color: '#94a3b8', background: 'none', border: 'none', cursor: 'pointer' }}>
        Cancel
      </button>
    </div>
  </div>
);

// ----- YAML Export/Import -----

export function workflowToYAML(workflow: Workflow): string {
  const lines: string[] = [
    `name: ${workflow.name}`,
    `description: ${workflow.description}`,
    `version: ${workflow.version}`,
    `status: ${workflow.status}`,
    `nodes:`,
  ];

  for (const node of workflow.nodes) {
    lines.push(`  - id: ${node.id}`);
    lines.push(`    type: ${node.type}`);
    lines.push(`    name: ${node.name}`);
    lines.push(`    position: { x: ${node.position.x}, y: ${node.position.y} }`);
    if (Object.keys(node.config).length > 0) {
      lines.push(`    config: ${JSON.stringify(node.config)}`);
    }
  }

  lines.push(`edges:`);
  for (const edge of workflow.edges) {
    lines.push(`  - source: ${edge.sourceNodeId}`);
    lines.push(`    target: ${edge.targetNodeId}`);
    if (edge.label) lines.push(`    label: ${edge.label}`);
  }

  return lines.join('\n');
}

export function yamlToWorkflow(yaml: string): Partial<Workflow> {
  const lines = yaml.split('\n');
  const workflow: Partial<Workflow> = { nodes: [], edges: [] };

  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.startsWith('name:')) workflow.name = trimmed.slice(5).trim();
    if (trimmed.startsWith('description:')) workflow.description = trimmed.slice(12).trim();
  }

  return workflow;
}

// ----- Main FlowBuilder Component -----

interface FlowBuilderProps {
  workflow?: Workflow;
  executions?: ExecutionVisualization[];
  onSave?: (workflow: Workflow) => void;
  readOnly?: boolean;
}

const FlowBuilder: React.FC<FlowBuilderProps> = ({ workflow: initial, executions, onSave, readOnly }) => {
  const [nodes, setNodes] = useState<WorkflowNode[]>(initial?.nodes || []);
  const [edges, setEdges] = useState<WorkflowEdge[]>(initial?.edges || []);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [showTemplates, setShowTemplates] = useState(!initial);
  const [workflowName, setWorkflowName] = useState(initial?.name || 'New Workflow');
  const dragRef = useRef<{ nodeId: string; startX: number; startY: number; origX: number; origY: number } | null>(null);
  const canvasRef = useRef<HTMLDivElement>(null);

  const addNode = useCallback((type: NodeType) => {
    const info = NODE_TYPE_INFO.find((n) => n.type === type);
    const newNode: WorkflowNode = {
      id: `${type}-${Date.now()}`,
      workflowId: initial?.id || '',
      type,
      name: info?.label || type,
      config: {},
      position: { x: 200 + Math.random() * 300, y: 100 + Math.random() * 300 },
      timeoutSeconds: 30,
      retryCount: 0,
    };
    setNodes((prev) => [...prev, newNode]);
  }, [initial?.id]);

  const deleteNode = useCallback((nodeId: string) => {
    setNodes((prev) => prev.filter((n) => n.id !== nodeId));
    setEdges((prev) => prev.filter((e) => e.sourceNodeId !== nodeId && e.targetNodeId !== nodeId));
    setSelectedNode(null);
  }, []);

  const handleDragStart = useCallback((nodeId: string, e: React.MouseEvent) => {
    if (readOnly) return;
    const node = nodes.find((n) => n.id === nodeId);
    if (!node) return;
    dragRef.current = { nodeId, startX: e.clientX, startY: e.clientY, origX: node.position.x, origY: node.position.y };

    const handleMove = (me: MouseEvent) => {
      if (!dragRef.current) return;
      const dx = me.clientX - dragRef.current.startX;
      const dy = me.clientY - dragRef.current.startY;
      setNodes((prev) =>
        prev.map((n) =>
          n.id === dragRef.current!.nodeId
            ? { ...n, position: { x: dragRef.current!.origX + dx, y: dragRef.current!.origY + dy } }
            : n
        )
      );
    };

    const handleUp = () => {
      dragRef.current = null;
      window.removeEventListener('mousemove', handleMove);
      window.removeEventListener('mouseup', handleUp);
    };

    window.addEventListener('mousemove', handleMove);
    window.addEventListener('mouseup', handleUp);
  }, [nodes, readOnly]);

  const loadTemplate = useCallback((template: WorkflowTemplate) => {
    setNodes(template.nodes);
    setEdges(template.edges);
    setWorkflowName(template.name);
  }, []);

  const handleSave = useCallback(() => {
    if (!onSave) return;
    onSave({
      id: initial?.id || '',
      tenantId: initial?.tenantId || '',
      name: workflowName,
      description: initial?.description || '',
      status: initial?.status || 'draft',
      version: (initial?.version || 0) + 1,
      nodes,
      edges,
      maxTimeoutSeconds: initial?.maxTimeoutSeconds || 300,
      maxRetries: initial?.maxRetries || 3,
      totalExecutions: initial?.totalExecutions || 0,
      successRate: initial?.successRate || 0,
      createdAt: initial?.createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    });
  }, [initial, nodes, edges, workflowName, onSave]);

  const handleExportYAML = useCallback(() => {
    const yaml = workflowToYAML({
      id: initial?.id || '',
      tenantId: '',
      name: workflowName,
      description: '',
      status: 'draft',
      version: 1,
      nodes,
      edges,
      maxTimeoutSeconds: 300,
      maxRetries: 3,
      totalExecutions: 0,
      successRate: 0,
      createdAt: '',
      updatedAt: '',
    });
    const blob = new Blob([yaml], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${workflowName.replace(/\s+/g, '-').toLowerCase()}.yaml`;
    a.click();
    URL.revokeObjectURL(url);
  }, [nodes, edges, workflowName, initial?.id]);

  return (
    <div style={{ display: 'flex', height: '100%', background: '#0f172a', color: '#e2e8f0', fontFamily: 'sans-serif' }}>
      {!readOnly && <NodePalette onAddNode={addNode} />}

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Toolbar */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 12px', background: '#1e293b', borderBottom: '1px solid #334155' }}>
          <input
            value={workflowName}
            onChange={(e) => setWorkflowName(e.target.value)}
            readOnly={readOnly}
            style={{ background: 'transparent', border: 'none', color: '#e2e8f0', fontSize: '16px', fontWeight: 600, outline: 'none', flex: 1 }}
          />
          <button onClick={() => setShowTemplates(true)} style={toolbarBtn}>Templates</button>
          <button onClick={handleExportYAML} style={toolbarBtn}>Export YAML</button>
          {selectedNode && !readOnly && (
            <button onClick={() => deleteNode(selectedNode)} style={{ ...toolbarBtn, borderColor: '#ef4444' }}>Delete Node</button>
          )}
          {onSave && !readOnly && (
            <button onClick={handleSave} style={{ ...toolbarBtn, background: '#3b82f6', borderColor: '#3b82f6' }}>Save</button>
          )}
        </div>

        {/* Canvas */}
        <div ref={canvasRef} style={{ flex: 1, position: 'relative', overflow: 'auto' }}>
          <svg style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', pointerEvents: 'none' }}>
            {edges.map((edge) => {
              const source = nodes.find((n) => n.id === edge.sourceNodeId);
              const target = nodes.find((n) => n.id === edge.targetNodeId);
              if (!source || !target) return null;
              return <FlowEdge key={edge.id} edge={edge} sourcePos={source.position} targetPos={target.position} />;
            })}
          </svg>

          {nodes.map((node) => (
            <FlowNode
              key={node.id}
              node={node}
              selected={selectedNode === node.id}
              execution={executions?.find((e) => e.nodeId === node.id)}
              onSelect={setSelectedNode}
              onDragStart={handleDragStart}
            />
          ))}

          {nodes.length === 0 && (
            <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <div style={{ textAlign: 'center', color: '#64748b' }}>
                <div style={{ fontSize: '48px', marginBottom: '12px' }}>🔧</div>
                <div style={{ fontSize: '16px' }}>Drag nodes from the palette or choose a template</div>
              </div>
            </div>
          )}
        </div>
      </div>

      {showTemplates && <TemplateSelector onSelect={loadTemplate} onClose={() => setShowTemplates(false)} />}
    </div>
  );
};

const toolbarBtn: React.CSSProperties = {
  padding: '6px 12px',
  background: 'transparent',
  border: '1px solid #334155',
  borderRadius: '6px',
  color: '#e2e8f0',
  fontSize: '12px',
  cursor: 'pointer',
};

export default FlowBuilder;
