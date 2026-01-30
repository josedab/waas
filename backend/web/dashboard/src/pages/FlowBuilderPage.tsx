import { useState, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/services/api';

interface WorkflowNode {
  id: string;
  type: string;
  label: string;
  config: Record<string, unknown>;
  position: { x: number; y: number };
}

interface WorkflowEdge {
  id: string;
  source_node_id: string;
  target_node_id: string;
  condition: string;
}

interface Workflow {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  status: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  created_at: string;
  updated_at: string;
}

interface WorkflowsResponse {
  workflows: Workflow[];
  total: number;
}

const NODE_TYPES = [
  { type: 'trigger', label: 'Trigger', icon: '⚡', color: 'bg-yellow-100 border-yellow-400' },
  { type: 'transform', label: 'Transform', icon: '🔄', color: 'bg-blue-100 border-blue-400' },
  { type: 'filter', label: 'Filter', icon: '🔍', color: 'bg-green-100 border-green-400' },
  { type: 'http', label: 'HTTP Call', icon: '🌐', color: 'bg-purple-100 border-purple-400' },
  { type: 'condition', label: 'Condition', icon: '❓', color: 'bg-orange-100 border-orange-400' },
  { type: 'delay', label: 'Delay', icon: '⏱️', color: 'bg-gray-100 border-gray-400' },
  { type: 'split', label: 'Fan Out', icon: '🔀', color: 'bg-pink-100 border-pink-400' },
  { type: 'notify', label: 'Notify', icon: '🔔', color: 'bg-red-100 border-red-400' },
];

export function FlowBuilderPage() {
  const queryClient = useQueryClient();
  const [selectedWorkflow, setSelectedWorkflow] = useState<Workflow | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newWorkflowName, setNewWorkflowName] = useState('');
  const [newWorkflowDesc, setNewWorkflowDesc] = useState('');
  const [draggedNodeType, setDraggedNodeType] = useState<string | null>(null);

  const { data: workflowsData, isLoading } = useQuery({
    queryKey: ['workflows'],
    queryFn: () => apiClient.get<WorkflowsResponse>('/flow-builder/workflows'),
  });

  const createWorkflow = useMutation({
    mutationFn: (data: { name: string; description: string }) =>
      apiClient.post('/flow-builder/workflows', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workflows'] });
      setShowCreateModal(false);
      setNewWorkflowName('');
      setNewWorkflowDesc('');
    },
  });

  const deleteWorkflow = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/flow-builder/workflows/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workflows'] });
      setSelectedWorkflow(null);
    },
  });

  const addNode = useMutation({
    mutationFn: ({ workflowId, node }: { workflowId: string; node: Partial<WorkflowNode> }) =>
      apiClient.put(`/flow-builder/workflows/${workflowId}`, {
        nodes: [...(selectedWorkflow?.nodes || []), node],
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['workflows'] }),
  });

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      if (!draggedNodeType || !selectedWorkflow) return;

      const rect = e.currentTarget.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;

      const nodeType = NODE_TYPES.find(n => n.type === draggedNodeType);
      addNode.mutate({
        workflowId: selectedWorkflow.id,
        node: {
          type: draggedNodeType,
          label: nodeType?.label || draggedNodeType,
          config: {},
          position: { x, y },
        },
      });
      setDraggedNodeType(null);
    },
    [draggedNodeType, selectedWorkflow, addNode]
  );

  const workflows: Workflow[] = workflowsData?.workflows || [];

  return (
    <div className="h-full flex">
      {/* Sidebar: Workflow list + Node palette */}
      <div className="w-72 bg-white border-r border-gray-200 flex flex-col">
        {/* Workflow list */}
        <div className="p-4 border-b border-gray-200">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold text-gray-900">Workflows</h2>
            <button
              onClick={() => setShowCreateModal(true)}
              className="px-3 py-1 bg-indigo-600 text-white text-sm rounded-md hover:bg-indigo-700"
            >
              + New
            </button>
          </div>

          {isLoading ? (
            <p className="text-sm text-gray-500">Loading...</p>
          ) : workflows.length === 0 ? (
            <p className="text-sm text-gray-500">No workflows yet</p>
          ) : (
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {workflows.map((wf: Workflow) => (
                <div
                  key={wf.id}
                  onClick={() => setSelectedWorkflow(wf)}
                  className={`p-2 rounded-md cursor-pointer text-sm ${
                    selectedWorkflow?.id === wf.id
                      ? 'bg-indigo-50 border border-indigo-200'
                      : 'hover:bg-gray-50 border border-transparent'
                  }`}
                >
                  <div className="font-medium text-gray-900">{wf.name}</div>
                  <div className="text-xs text-gray-500">
                    {wf.status} · {wf.nodes?.length || 0} nodes
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Node palette */}
        <div className="p-4 flex-1">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">Node Types</h3>
          <p className="text-xs text-gray-500 mb-3">Drag nodes onto the canvas</p>
          <div className="space-y-2">
            {NODE_TYPES.map(nodeType => (
              <div
                key={nodeType.type}
                draggable
                onDragStart={() => setDraggedNodeType(nodeType.type)}
                className={`p-2 border rounded-md cursor-grab active:cursor-grabbing ${nodeType.color} text-sm flex items-center gap-2`}
              >
                <span>{nodeType.icon}</span>
                <span>{nodeType.label}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Main canvas */}
      <div className="flex-1 flex flex-col bg-gray-50">
        {/* Toolbar */}
        <div className="bg-white border-b border-gray-200 px-4 py-2 flex items-center justify-between">
          <div>
            {selectedWorkflow ? (
              <div>
                <h1 className="text-lg font-semibold text-gray-900">{selectedWorkflow.name}</h1>
                <p className="text-xs text-gray-500">{selectedWorkflow.description}</p>
              </div>
            ) : (
              <h1 className="text-lg font-semibold text-gray-500">Select or create a workflow</h1>
            )}
          </div>
          {selectedWorkflow && (
            <div className="flex items-center gap-2">
              <span
                className={`px-2 py-1 text-xs rounded-full ${
                  selectedWorkflow.status === 'active'
                    ? 'bg-green-100 text-green-800'
                    : 'bg-gray-100 text-gray-800'
                }`}
              >
                {selectedWorkflow.status}
              </span>
              <button
                onClick={() => deleteWorkflow.mutate(selectedWorkflow.id)}
                className="px-3 py-1 text-sm text-red-600 hover:bg-red-50 rounded-md"
              >
                Delete
              </button>
            </div>
          )}
        </div>

        {/* Canvas */}
        <div
          className="flex-1 relative overflow-auto"
          onDragOver={e => e.preventDefault()}
          onDrop={handleDrop}
        >
          {selectedWorkflow ? (
            <div className="p-8 min-h-full">
              {(!selectedWorkflow.nodes || selectedWorkflow.nodes.length === 0) ? (
                <div className="flex items-center justify-center h-full text-gray-400">
                  <div className="text-center">
                    <p className="text-lg">Drag nodes here to build your workflow</p>
                    <p className="text-sm mt-1">Start with a Trigger node</p>
                  </div>
                </div>
              ) : (
                <div className="relative" style={{ minHeight: '600px' }}>
                  {selectedWorkflow.nodes.map((node: WorkflowNode) => (
                    <div
                      key={node.id}
                      className={`absolute p-3 rounded-lg border-2 shadow-sm min-w-[140px] ${
                        NODE_TYPES.find(n => n.type === node.type)?.color || 'bg-white border-gray-300'
                      }`}
                      style={{
                        left: node.position?.x || 0,
                        top: node.position?.y || 0,
                      }}
                    >
                      <div className="flex items-center gap-2">
                        <span>{NODE_TYPES.find(n => n.type === node.type)?.icon || '📦'}</span>
                        <span className="font-medium text-sm">{node.label}</span>
                      </div>
                      <div className="text-xs text-gray-500 mt-1">{node.type}</div>
                    </div>
                  ))}

                  {/* Render edges as SVG lines */}
                  <svg className="absolute inset-0 w-full h-full pointer-events-none">
                    {(selectedWorkflow.edges || []).map((edge: WorkflowEdge) => {
                      const source = selectedWorkflow.nodes.find(n => n.id === edge.source_node_id);
                      const target = selectedWorkflow.nodes.find(n => n.id === edge.target_node_id);
                      if (!source || !target) return null;
                      return (
                        <line
                          key={edge.id}
                          x1={(source.position?.x || 0) + 70}
                          y1={(source.position?.y || 0) + 30}
                          x2={(target.position?.x || 0) + 70}
                          y2={(target.position?.y || 0)}
                          stroke="#6366f1"
                          strokeWidth="2"
                          markerEnd="url(#arrowhead)"
                        />
                      );
                    })}
                    <defs>
                      <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
                        <polygon points="0 0, 10 3.5, 0 7" fill="#6366f1" />
                      </marker>
                    </defs>
                  </svg>
                </div>
              )}
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-gray-400">
              <p>Select a workflow from the sidebar or create a new one</p>
            </div>
          )}
        </div>
      </div>

      {/* Create workflow modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-96">
            <h2 className="text-lg font-semibold mb-4">Create Workflow</h2>
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input
                  type="text"
                  value={newWorkflowName}
                  onChange={e => setNewWorkflowName(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                  placeholder="My Workflow"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
                <textarea
                  value={newWorkflowDesc}
                  onChange={e => setNewWorkflowDesc(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                  rows={3}
                  placeholder="Describe your workflow..."
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-md"
              >
                Cancel
              </button>
              <button
                onClick={() =>
                  createWorkflow.mutate({ name: newWorkflowName, description: newWorkflowDesc })
                }
                disabled={!newWorkflowName}
                className="px-4 py-2 text-sm bg-indigo-600 text-white rounded-md hover:bg-indigo-700 disabled:opacity-50"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default FlowBuilderPage;
