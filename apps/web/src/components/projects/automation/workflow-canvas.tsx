import {
	Background,
	type Connection,
	Controls,
	type Edge,
	Handle,
	type Node,
	type NodeProps,
	Position,
	ReactFlow,
	ReactFlowProvider,
	useNodesState,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { AlertCircle, Trash2, X } from "lucide-react";
import {
	type CSSProperties,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { TaskCard } from "@/components/projects/interactions/task-card";
import type { TaskFieldUpdate } from "@/components/projects/interactions/view-utils";
import type { Task } from "@/lib/interaction-api";
import type { ProjectMember, TaskStatus, TaskType } from "@/lib/project-api";
import { cn } from "@/lib/utils";
import type { WorkflowEdge, WorkflowNode } from "@/lib/workflow-api";

// Kanban columns already convey status, so the board card's default visible
// fields omit it — but the canvas has no such grouping, so status must be
// shown (and editable) explicitly here.
const WORKFLOW_NODE_VISIBLE_FIELDS = [
	"assignee",
	"status",
	"type",
	"story_points",
];

interface TaskNodeData extends Record<string, unknown> {
	task?: Task;
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	members: ProjectMember[];
	taskIdPrefix: string;
	canEdit: boolean;
	canEditTask: boolean;
	onDelete: () => void;
	onUpdateTask: (taskId: string, payload: TaskFieldUpdate) => void;
}

function TaskNode({ data }: NodeProps<Node<TaskNodeData>>) {
	const { t } = useTranslation("projects");
	const {
		task,
		statuses,
		taskTypes,
		members,
		taskIdPrefix,
		canEdit,
		canEditTask,
		onDelete,
		onUpdateTask,
	} = data;

	if (!task) {
		return (
			<div className="w-64 rounded-xl border border-border/40 bg-card px-3 py-2.5 shadow-sm text-sm text-muted-foreground">
				{t("automation.canvas.unknownTask")}
			</div>
		);
	}

	return (
		<div className="group relative w-64">
			<Handle
				type="target"
				position={Position.Top}
				className="size-2.5! bg-primary/50! border-2! border-background!"
			/>
			<Handle
				type="source"
				position={Position.Bottom}
				className="size-2.5! bg-primary/50! border-2! border-background!"
			/>
			<TaskCard
				task={task}
				taskIdPrefix={taskIdPrefix}
				statuses={statuses}
				taskTypes={taskTypes}
				members={members}
				visibleFields={WORKFLOW_NODE_VISIBLE_FIELDS}
				canEdit={canEditTask}
				draggable={false}
				onUpdate={onUpdateTask}
			/>
			{canEdit && (
				<button
					type="button"
					onClick={(e) => {
						e.stopPropagation();
						onDelete();
					}}
					title={t("automation.nodePanel.removeNode")}
					className="nodrag absolute -top-2 -right-2 opacity-0 group-hover:opacity-100 flex size-5 shrink-0 items-center justify-center rounded-full border border-border/40 bg-card text-muted-foreground/60 shadow-sm hover:text-destructive hover:bg-destructive/10 transition-all"
				>
					<Trash2 className="size-3" />
				</button>
			)}
		</div>
	);
}

const nodeTypes = { taskNode: TaskNode };

interface WorkflowCanvasProps {
	nodes: WorkflowNode[];
	edges: WorkflowEdge[];
	tasks: Task[];
	statuses: TaskStatus[];
	taskTypes: TaskType[];
	members: ProjectMember[];
	taskIdPrefix?: string;
	/** Governs graph-structure editing: add/remove/drag nodes, connect edges. */
	canEdit: boolean;
	/** Governs inline task-field editing (status/assignee) on the node card —
	 *  independent of `canEdit`, since tasks are normally updated while a
	 *  workflow is active, not just while its graph is still a draft. */
	canEditTask: boolean;
	onOpenTask: (taskId: string) => void;
	onConnect: (sourceNodeId: string, targetNodeId: string) => void;
	onMoveNode: (nodeId: string, posX: number, posY: number) => void;
	onDeleteNode: (nodeId: string) => void;
	onDeleteEdge: (edgeId: string) => void;
	onUpdateTask: (taskId: string, payload: TaskFieldUpdate) => void;
	errorMessage?: string | null;
	onDismissError?: () => void;
}

export function WorkflowCanvas({
	nodes,
	edges,
	tasks,
	statuses,
	taskTypes,
	members,
	taskIdPrefix = "",
	canEdit,
	canEditTask,
	onOpenTask,
	onConnect,
	onMoveNode,
	onDeleteNode,
	onDeleteEdge,
	onUpdateTask,
	errorMessage,
	onDismissError,
}: WorkflowCanvasProps) {
	const { t } = useTranslation("projects");
	const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null);

	const taskById = useMemo(
		() => new Map(tasks.map((task) => [task.id, task])),
		[tasks],
	);

	const flowNodes = useMemo<Node<TaskNodeData>[]>(
		() =>
			nodes.map((n) => {
				const task = taskById.get(n.task_id);
				return {
					id: n.id,
					type: "taskNode",
					position: { x: n.pos_x, y: n.pos_y },
					draggable: canEdit,
					connectable: canEdit,
					data: {
						task,
						statuses,
						taskTypes,
						members,
						taskIdPrefix,
						canEdit,
						canEditTask,
						onDelete: () => onDeleteNode(n.id),
						onUpdateTask,
					},
				};
			}),
		[
			nodes,
			taskById,
			statuses,
			taskTypes,
			members,
			taskIdPrefix,
			canEdit,
			canEditTask,
			onDeleteNode,
			onUpdateTask,
		],
	);

	// React Flow needs a controlled node array wired through onNodesChange for
	// drag/select/move interactions to update the rendered position live —
	// otherwise the node doesn't visually follow the cursor while dragging.
	const [rfNodes, setRfNodes, onNodesChange] =
		useNodesState<Node<TaskNodeData>>(flowNodes);
	const isDraggingRef = useRef(false);
	// Position a node was just dropped at, pending server confirmation. The
	// move mutation is async, so flowNodes briefly still reflects the old
	// pre-drag position on any re-render that happens before it resolves —
	// without this, that stale recompute snaps the node back for a beat.
	const pendingPositionsRef = useRef(
		new Map<string, { x: number; y: number }>(),
	);

	// Resync from server-derived data (nodes added/removed, task fields
	// updated elsewhere) but never mid-drag, or the gesture would be clobbered.
	// Any node with an unconfirmed drop keeps its locally-known position until
	// flowNodes reports that same position back (i.e. the server caught up).
	useEffect(() => {
		if (isDraggingRef.current) return;
		const pending = pendingPositionsRef.current;
		if (pending.size === 0) {
			setRfNodes(flowNodes);
			return;
		}
		setRfNodes(
			flowNodes.map((n) => {
				const p = pending.get(n.id);
				if (!p) return n;
				if (n.position.x === p.x && n.position.y === p.y) {
					pending.delete(n.id);
					return n;
				}
				return { ...n, position: p };
			}),
		);
	}, [flowNodes, setRfNodes]);

	const flowEdges = useMemo<Edge[]>(
		() =>
			edges.map((e) => ({
				id: e.id,
				source: e.source_node_id,
				target: e.target_node_id,
				animated: true,
				selected: e.id === selectedEdgeId,
				style:
					e.id === selectedEdgeId
						? { stroke: "var(--color-primary)", strokeWidth: 2.5 }
						: undefined,
			})),
		[edges, selectedEdgeId],
	);

	const handleConnect = useCallback(
		(connection: Connection) => {
			if (!canEdit || !connection.source || !connection.target) return;
			onConnect(connection.source, connection.target);
		},
		[canEdit, onConnect],
	);

	return (
		<div className="relative flex-1 min-h-0">
			{errorMessage && (
				<div className="absolute top-3 left-1/2 -translate-x-1/2 z-10 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive shadow-lg max-w-md">
					<AlertCircle className="size-3.5 shrink-0" />
					<span className="flex-1">{errorMessage}</span>
					{onDismissError && (
						<button type="button" onClick={onDismissError} className="shrink-0">
							<X className="size-3.5" />
						</button>
					)}
				</div>
			)}
			{selectedEdgeId && canEdit && (
				<div className="absolute top-3 right-3 z-10">
					<button
						type="button"
						onClick={() => {
							onDeleteEdge(selectedEdgeId);
							setSelectedEdgeId(null);
						}}
						className="flex items-center gap-1.5 rounded-lg border border-destructive/30 bg-card px-3 py-1.5 text-xs font-medium text-destructive shadow-lg hover:bg-destructive/10 transition-colors"
					>
						<Trash2 className="size-3.5" />
						{t("automation.canvas.deleteEdge")}
					</button>
				</div>
			)}
			<ReactFlowProvider>
				<ReactFlow
					nodes={rfNodes}
					edges={flowEdges}
					nodeTypes={nodeTypes}
					onNodesChange={onNodesChange}
					onNodeClick={(_, node) => {
						const taskId = (node.data as TaskNodeData).task?.id;
						if (taskId) onOpenTask(taskId);
					}}
					onNodeDragStart={() => {
						isDraggingRef.current = true;
					}}
					onNodeDragStop={(_, node) => {
						isDraggingRef.current = false;
						pendingPositionsRef.current.set(node.id, node.position);
						onMoveNode(node.id, node.position.x, node.position.y);
					}}
					onConnect={handleConnect}
					onEdgeClick={(_, edge) => setSelectedEdgeId(edge.id)}
					onPaneClick={() => setSelectedEdgeId(null)}
					nodesDraggable={canEdit}
					nodesConnectable={canEdit}
					elementsSelectable
					fitView
					proOptions={{ hideAttribution: true }}
					className={cn(!canEdit && "opacity-90")}
				>
					<Background gap={20} />
					<Controls
						showInteractive={false}
						style={
							{
								"--xy-controls-button-background-color": "var(--sidebar)",
								"--xy-controls-button-background-color-hover":
									"var(--sidebar-accent)",
								"--xy-controls-button-color": "var(--sidebar-foreground)",
								"--xy-controls-button-color-hover":
									"var(--sidebar-accent-foreground)",
								"--xy-controls-button-border-color": "var(--sidebar-border)",
							} as CSSProperties
						}
					/>
				</ReactFlow>
			</ReactFlowProvider>
		</div>
	);
}
