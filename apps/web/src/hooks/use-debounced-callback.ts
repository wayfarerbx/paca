import { useCallback, useRef } from "react";

export function useDebouncedCallback<Args extends unknown[]>(
	callback: (...args: Args) => void,
	delay: number,
): (...args: Args) => void {
	const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const callbackRef = useRef(callback);
	callbackRef.current = callback;

	return useCallback(
		(...args: Args) => {
			if (timerRef.current) clearTimeout(timerRef.current);
			timerRef.current = setTimeout(() => callbackRef.current(...args), delay);
		},
		[delay],
	);
}
