import { createFileRoute } from "@tanstack/react-router";
import { KanbanSquare, ListTodo, Sparkles } from "lucide-react";

import { Badge } from "@/components/ui/badge";

export const Route = createFileRoute(
	"/_authenticated/projects/$projectId/integrations/",
)({
	component: IntegrationsPage,
});

const FEATURES = [
	{
		icon: ListTodo,
		title: "Product Backlog",
		description:
			"Capture user stories, epics, and tasks. Prioritize with drag-and-drop and estimate with story points.",
		tag: "Backlog",
		color: "bg-blue-500/10 text-blue-500",
	},
	{
		icon: KanbanSquare,
		title: "Sprint Board",
		description:
			"Run time-boxed iterations with a Scrum board. Move work through To-Do, In Progress, and Done columns.",
		tag: "Sprints",
		color: "bg-emerald-500/10 text-emerald-500",
	},
	{
		icon: Sparkles,
		title: "AI-Powered Planning",
		description:
			"Let AI agents groom your backlog, suggest sprint goals, and auto-estimate story complexity.",
		tag: "AI",
		color: "bg-amber-500/10 text-amber-500",
	},
] as const;

function IntegrationsPage() {
	return (
		<div className="flex flex-col">
			<div className="relative overflow-hidden border-b border-border/30">
				<div
					className="pointer-events-none absolute inset-0 opacity-60"
					style={{
						backgroundImage:
							"radial-gradient(circle, color-mix(in oklch, var(--color-primary) 12%, transparent) 1px, transparent 1px)",
						backgroundSize: "20px 20px",
						maskImage:
							"radial-gradient(ellipse 70% 100% at 0% 0%, black 20%, transparent 70%)",
					}}
				/>
				<div className="relative px-8 py-8">
					<div className="mb-2.5 flex items-center gap-2">
						<Badge
							variant="secondary"
							className="gap-1.5 px-2.5 py-1 text-[11px] font-bold border border-border/30"
						>
							<span className="size-1.5 rounded-full bg-secondary inline-block" />
							Coming Soon
						</Badge>
					</div>
					<h1 className="font-[Syne] text-[26px] font-bold tracking-tight">
						Integrations
					</h1>
					<p className="mt-1.5 max-w-lg text-[14px] text-muted-foreground">
						Product backlog and sprint management — the core of your scrumban
						workflow.
					</p>
				</div>
			</div>

			<div className="px-8 py-7">
				<div className="rounded-2xl border-2 border-dashed border-border/25 bg-muted/10 p-10 text-center mb-6 hover:border-border/40 hover:bg-muted/15 transition-all duration-200">
					<div className="flex items-center justify-center gap-3 mb-4">
						<div className="flex size-12 items-center justify-center rounded-xl bg-blue-500/10">
							<ListTodo className="size-6 text-blue-500" />
						</div>
						<div className="h-px flex-1 max-w-8 bg-linear-to-r from-border/40 to-transparent" />
						<div className="flex size-12 items-center justify-center rounded-xl bg-emerald-500/10">
							<KanbanSquare className="size-6 text-emerald-500" />
						</div>
					</div>
					<h2 className="font-[Syne] text-[20px] font-bold tracking-tight">
						Backlog & Sprint Board
					</h2>
					<p className="mt-2 max-w-sm mx-auto text-[13px] text-muted-foreground/70 leading-relaxed">
						Full scrumban workflow with product backlog grooming, sprint
						planning, and AI-assisted estimation.
					</p>
				</div>

				<div className="grid gap-4 sm:grid-cols-3">
					{FEATURES.map(({ icon: Icon, title, description, tag, color }) => (
						<div
							key={title}
							className="rounded-xl border border-border/25 bg-muted/15 p-5 transition-all duration-150 hover:bg-muted/25 hover:border-border/35"
						>
							<div className="flex items-start justify-between mb-3">
								<div
									className={`flex size-9 items-center justify-center rounded-lg ${color}`}
								>
									<Icon className="size-4" />
								</div>
								<Badge
									variant="outline"
									className="text-[10px] font-bold border-border/30"
								>
									{tag}
								</Badge>
							</div>
							<p className="text-[14px] font-semibold text-foreground">
								{title}
							</p>
							<p className="mt-1 text-[12px] text-muted-foreground/70 leading-relaxed">
								{description}
							</p>
						</div>
					))}
				</div>
			</div>
		</div>
	);
}
