import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import { RouteErrorComponent } from "@/components/route-error-boundary";

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()(
	{
		component: () => (
			<>
				<Outlet />
				{import.meta.env.DEV && <TanStackRouterDevtools />}
			</>
		),
		// Global safety net: without this, any uncaught loader or render error
		// in a lazy-loaded route crashes with "Element type is invalid… Check
		// the render method of Lazy" (TanStack Router's internal lazy wrapper).
		defaultErrorComponent: ({ error }) => <RouteErrorComponent error={error} />,
	},
);
