import { GitBranch, ShieldCheck, Users } from "lucide-react";

const FEATURES = [
	{
		icon: Users,
		title: "Fluid Roles",
		desc: "From PO and BA to Dev and QA, work can move between humans and AI agents seamlessly.",
	},
	{
		icon: GitBranch,
		title: "Contextual Assignment",
		desc: "Tasks are assigned by strengths: AI for speed and precision, humans for judgment and creativity.",
	},
	{
		icon: ShieldCheck,
		title: "Human-in-Control",
		desc: "Every AI contribution stays transparent and supervised so teams remain the final decision-makers.",
	},
] as const;

import { GitHubIcon } from "@/components/icons/github-icon";

export function BrandPanel() {
	return (
		<div className="relative hidden flex-col justify-between overflow-hidden rounded-l-3xl p-10 lg:flex">
			{/* Base gradient — deep navy */}
			<div className="pointer-events-none absolute inset-0 bg-[#091830]" />
			<div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_80%_60%_at_10%_0%,#1b3d6e,transparent)] opacity-80" />
			<div className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_60%_50%_at_90%_100%,#0b2040,transparent)]" />

			{/* Subtle grid texture */}
			<div
				className="pointer-events-none absolute inset-0 opacity-[0.055]"
				style={{
					backgroundImage:
						"linear-gradient(rgba(255,255,255,0.6) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.6) 1px, transparent 1px)",
					backgroundSize: "36px 36px",
				}}
			/>

			{/* Green ambient glow — top left */}
			<div className="pointer-events-none absolute -left-24 -top-24 h-72 w-72 rounded-full bg-[radial-gradient(circle,rgba(50,205,50,0.16),transparent_60%)]" />

			{/* Blue depth glow — bottom right */}
			<div className="pointer-events-none absolute -bottom-20 -right-10 h-80 w-80 rounded-full bg-[radial-gradient(circle,rgba(46,73,128,0.55),transparent_55%)]" />

			{/* Decorative concentric rings — right side */}
			<div className="pointer-events-none absolute right-0 top-1/2 h-105 w-105 -translate-y-1/2 translate-x-[42%] rounded-full border border-white/6" />
			<div className="pointer-events-none absolute right-0 top-1/2 h-70 w-70 -translate-y-1/2 translate-x-[42%] rounded-full border border-white/8" />

			<div className="relative">
				{/* Logo + brand */}
				<div className="mb-8 flex items-center gap-3">
					<div className="flex size-9 shrink-0 items-center justify-center rounded-xl border border-white/15 bg-white/8 shadow-lg shadow-black/20 backdrop-blur-sm">
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
					<span
						className="bg-clip-text text-transparent"
						style={{
							backgroundImage:
								"linear-gradient(90deg, #32cd32 0%, #7de87d 100%)",
						}}
					>
						human and AI.
					</span>
				</h2>
				<p className="mb-8 text-sm leading-relaxed text-white/55">
					Paca is the open-source collaborative task management engine where
					human creativity and AI efficiency work together in a shared Scrumban
					workflow.
				</p>

				{/* Feature cards */}
				<ul className="space-y-2.5">
					{FEATURES.map(({ icon: Icon, title, desc }) => (
						<li
							key={title}
							className="flex items-start gap-3.5 rounded-xl border border-white/8 bg-white/4 px-4 py-3.5 transition-colors hover:border-white/[0.14] hover:bg-white/[0.07]"
						>
							<div className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-lg bg-[rgba(50,205,50,0.14)] ring-1 ring-[rgba(50,205,50,0.22)]">
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
