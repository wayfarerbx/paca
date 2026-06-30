import { AlertCircle } from "lucide-react";
import {
	Component,
	type ComponentType,
	lazy,
	type ReactNode,
	Suspense,
} from "react";
import { useTranslation } from "react-i18next";
import type { PluginRegistration } from "@/lib/plugin-api";

// ── Module Federation dynamic import ─────────────────────────────────────────

/**
 * Cache of lazy-loaded remote components keyed by `remoteEntryUrl#component`.
 * We memo-ize so each unique (url, component) pair is only loaded once per
 * browser session regardless of how many extension points use it.
 */
const componentCache = new Map<string, ReturnType<typeof lazy>>();

/**
 * Initialize the share scope with shared dependencies.
 * This ensures that remote components receive the same instances of React
 * and other shared libraries as the host application.
 */
async function initializeShareScope(): Promise<Record<string, unknown>> {
	// Ensure the global share scope object exists
	if (!globalThis.__federation_shared__) {
		globalThis.__federation_shared__ = {};
	}
	if (!globalThis.__federation_shared__.default) {
		globalThis.__federation_shared__.default = {};
	}

	const defaultScope = globalThis.__federation_shared__.default;

	// Dynamically import the shared modules
	const [react, reactDom, reactQuery] = await Promise.all([
		import("react"),
		import("react-dom"),
		import("@tanstack/react-query").catch(() => null),
	]);

	// Debug logging
	console.log("[Host] Initializing share scope with modules:", {
		react: !!react,
		"react-dom": !!reactDom,
		"@tanstack/react-query": !!reactQuery,
	});

	// Register React in the share scope
	defaultScope.react = {
		"19.0.0": {
			get: () => Promise.resolve(() => Promise.resolve(react)),
			version: "19.0.0",
		},
	} as Record<string, unknown>;

	// Register React DOM in the share scope
	defaultScope["react-dom"] = {
		"19.0.0": {
			get: () => Promise.resolve(() => Promise.resolve(reactDom)),
			version: "19.0.0",
		},
	} as Record<string, unknown>;

	// Register React Query in the share scope if available
	if (reactQuery) {
		defaultScope["@tanstack/react-query"] = {
			"5.0.0": {
				get: () => Promise.resolve(() => Promise.resolve(reactQuery)),
				version: "5.0.0",
			},
		} as Record<string, unknown>;
	}

	console.log("[Host] Share scope initialized:", Object.keys(defaultScope));
	return defaultScope;
}

function getRemoteComponent(remoteEntryUrl: string, component: string) {
	const cacheKey = `${remoteEntryUrl}#${component}`;
	const cached = componentCache.get(cacheKey);
	if (cached) return cached;

	const componentPromise = lazy(async () => {
		console.log(
			`[Host] Loading remote component: ${component} from ${remoteEntryUrl}`,
		);

		// CRITICAL: Initialize the share scope BEFORE loading the container
		// The remote module's top-level await importShared() executes immediately
		// when the module is loaded, so the share scope must be ready first
		const shareScope = await initializeShareScope();
		console.log("[Host] Share scope initialized before container loading");

		// Dynamically import the remote entry ESM and use its get/init exports.
		const container = await loadRemoteContainer(remoteEntryUrl);
		console.log("[Host] Container loaded:", !!container);

		// Initialize the container with the share scope
		console.log("[Host] Initializing container with share scope");
		await container.init(shareScope);
		console.log("[Host] Container initialized");

		console.log(`[Host] Getting component factory for: ./${component}`);
		const factory = await container.get(`./${component}`);
		console.log("[Host] Component factory obtained:", !!factory);

		const mod = (await factory()) as
			| { default: ComponentType<unknown> }
			| ComponentType<unknown>;
		console.log("[Host] Module loaded:", mod);

		// Handle both module object with default export and direct component export
		if (mod && typeof mod === "object" && "default" in mod) {
			console.log("[Host] Returning module with default export");
			return mod;
		}
		// If mod is the component itself, wrap it in a module object
		console.log("[Host] Wrapping component in module object");
		return { default: mod as ComponentType<unknown> };
	});

	componentCache.set(cacheKey, componentPromise);
	return componentPromise;
}

// ── Remote container loader ───────────────────────────────────────────────────

interface RemoteContainer {
	init(shareScope: Record<string, unknown>): Promise<void>;
	get(module: string): Promise<() => Record<string, unknown>>;
}

const containerCache = new Map<string, Promise<RemoteContainer>>();

function loadRemoteContainer(remoteEntryUrl: string): Promise<RemoteContainer> {
	const cached = containerCache.get(remoteEntryUrl);
	if (cached) return cached;

	// @module-federation/vite emits ES modules that export { get, init }
	// directly. Use dynamic import() to load the module and get its exports.
	const containerPromise =
		// biome-ignore lint/suspicious/noExplicitAny: dynamic remote import
		(import(/* @vite-ignore */ remoteEntryUrl) as Promise<any>).then((mod) => {
			const container: RemoteContainer = mod.default ?? mod;
			if (
				typeof container.get !== "function" ||
				typeof container.init !== "function"
			) {
				throw new Error(
					`Remote entry at ${remoteEntryUrl} does not export a valid container (missing get/init)`,
				);
			}
			return container;
		});

	containerCache.set(remoteEntryUrl, containerPromise);
	return containerPromise;
}

// globalThis share scope injected by Vite's federation plugin at build time.
declare const __webpack_share_scopes__:
	| { default: Record<string, unknown> }
	| undefined;

// Extend globalThis to include the federation shared scope
declare global {
	var __federation_shared__:
		| Record<string, Record<string, unknown>>
		| undefined;
}

// ── Error boundary ────────────────────────────────────────────────────────────

interface ErrorBoundaryState {
	hasError: boolean;
	message: string;
}

interface ErrorBoundaryProps {
	pluginName: string;
	children: ReactNode;
	fallback?: ReactNode;
}

class PluginErrorBoundary extends Component<
	ErrorBoundaryProps,
	ErrorBoundaryState
> {
	constructor(props: ErrorBoundaryProps) {
		super(props);
		this.state = { hasError: false, message: "" };
	}

	static getDerivedStateFromError(error: unknown): ErrorBoundaryState {
		return {
			hasError: true,
			message: error instanceof Error ? error.message : String(error),
		};
	}

	render() {
		if (this.state.hasError) {
			return this.props.fallback ?? null;
		}
		return this.props.children;
	}
}

// ── Public component ──────────────────────────────────────────────────────────

export interface RemoteComponentProps {
	registration: PluginRegistration;
	/** Props forwarded to the remote component */
	// biome-ignore lint/suspicious/noExplicitAny: remote component props are untyped
	componentProps?: Record<string, any>;
	fallback?: ReactNode;
}

/**
 * Loads and renders a single remote (plugin) component via Module Federation.
 * Each remote component is individually wrapped in a Suspense + ErrorBoundary
 * so a failing plugin cannot break the host application.
 */
export function RemoteComponent({
	registration,
	componentProps,
	fallback,
}: RemoteComponentProps) {
	const { t } = useTranslation("errors");
	const LazyComponent = getRemoteComponent(
		registration.remoteEntryUrl,
		registration.component,
	);

	const defaultFallback = (
		<div className="flex items-center gap-2 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive">
			<AlertCircle className="size-3.5 shrink-0" />
			<span>
				{t("pluginLoadFailedPrefix")} <strong>{registration.pluginName}</strong>{" "}
				{t("pluginLoadFailedSuffix")}
			</span>
		</div>
	);

	return (
		<PluginErrorBoundary
			pluginName={registration.pluginName}
			fallback={fallback ?? defaultFallback}
		>
			<Suspense fallback={null}>
				<LazyComponent {...(componentProps ?? {})} />
			</Suspense>
		</PluginErrorBoundary>
	);
}
