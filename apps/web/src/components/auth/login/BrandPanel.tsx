import { BookOpen, Bot, RefreshCcw } from "lucide-react";
import { GitHubIcon } from "@/components/icons/github-icon";

const FEATURES = [
	{
		icon: Bot,
		title: "AI Agents as Peers",
		desc: "AI Agents join your Scrum team as first-class members — not plugins — picking tasks from the same backlog as humans.",
	},
	{
		icon: BookOpen,
		title: "BDD & SDD Hub",
		desc: "Align POs, BAs, and Agents through Gherkin scenarios and System Design Docs that keep the whole team in sync.",
	},
	{
		icon: RefreshCcw,
		title: "The P-A-C-A Cycle",
		desc: "Plan → Act → Check → Adapt. A Human-in-the-loop sprint loop built for complex, emergent projects where AI amplifies the team.",
	},
] as const;

export function BrandPanel() {
	return (
		<div className="relative hidden flex-col justify-between overflow-hidden rounded-l-xl bg-[#0a0a0a] p-10 lg:flex">
			{/* Lime ambient glow — top */}
			<div className="pointer-events-none absolute -left-24 -top-24 h-72 w-72 rounded-full bg-[radial-gradient(circle,rgba(158,217,87,0.08),transparent_60%)]" />

			{/* Decorative concentric rings — right side */}
			<div className="pointer-events-none absolute right-0 top-1/2 h-105 w-105 -translate-y-1/2 translate-x-[42%] rounded-full border border-white/5" />
			<div className="pointer-events-none absolute right-0 top-1/2 h-70 w-70 -translate-y-1/2 translate-x-[42%] rounded-full border border-white/7" />

			<div className="relative">
				{/* Logo + brand */}
				<div className="mb-8 flex items-center gap-3">
					<div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-white/10 bg-white/6 shadow-sm shadow-black/40">
						<img
							src="/paca-logo-dark.svg"
							alt="Paca logo"
							width={127}
							height={175}
							className="h-auto w-5 brightness-0 invert"
						/>
					</div>
					<span className="text-xl font-bold tracking-tight text-white">
						paca
					</span>
					<span className="rounded-full border border-white/20 bg-white/8 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-widest text-white/60">
						OSS
					</span>
				</div>

				<h2 className="display-title mb-3 text-[1.85rem] font-bold leading-tight text-balance text-white">
					One team, one board,{" "}
					<span className="text-[#9ed957]">human and AI.</span>
				</h2>
				<p className="mb-8 text-sm leading-relaxed text-white/55">
					Paca is the open-source project management engine designed for Complex
					Projects — where AI Agents and humans collaborate on the same Scrumban
					board as equal, first-class teammates.
				</p>

				{/* Feature cards */}
				<ul className="space-y-2.5">
					{FEATURES.map(({ icon: Icon, title, desc }) => (
						<li
							key={title}
							className="flex items-start gap-3.5 rounded-xl border border-white/8 bg-white/4 px-4 py-3.5 transition-colors hover:border-white/[0.14] hover:bg-white/[0.07]"
						>
							<div className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-lg bg-[rgba(158,217,87,0.12)] ring-1 ring-[rgba(158,217,87,0.2)]">
								<Icon className="size-3.5 text-(--palm)" />
							</div>
							<div>
								<p className="text-sm font-semibold text-white/90">{title}</p>
								<p className="mt-0.5 text-xs leading-relaxed text-white/50">
									{desc}
								</p>
							</div>
						</li>
					))}
				</ul>
			</div>

			{/* Footer */}
			<div className="relative mt-8 pt-6">
				{/* Gradient separator */}
				<div className="absolute inset-x-0 top-0 h-px bg-[linear-gradient(90deg,transparent,rgba(255,255,255,0.18),transparent)]" />

				<div className="flex items-center justify-between">
					<a
						href="https://github.com/Paca-AI/paca"
						target="_blank"
						rel="noopener noreferrer"
						className="inline-flex items-center gap-2 rounded-lg border border-white/15 bg-white/6 px-3.5 py-2 text-xs font-medium text-white! transition-all hover:border-white/25 hover:bg-white/12 hover:text-white!"
					>
						<GitHubIcon className="size-3.5" />
						View on GitHub
					</a>
					<p className="text-[11px] text-white/30">Apache-2.0 · Open Source</p>
				</div>
			</div>
		</div>
	);
}
