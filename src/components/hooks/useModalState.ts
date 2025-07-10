import { useState, useCallback } from "react";
import type { AppState, ModalMode } from "../types.js";

export function useModalState() {
	const [state, setState] = useState<AppState>({
		mode: "normal",
		selectedWorktreeIndex: 0,
		searchQuery: "",
		showHelp: false,
	});

	const setMode = useCallback((mode: ModalMode) => {
		setState((prev) => ({
			...prev,
			mode,
			searchQuery: mode === "filter" ? prev.searchQuery : "",
		}));
	}, []);

	const setSelectedIndex = useCallback((index: number) => {
		setState((prev) => ({ ...prev, selectedWorktreeIndex: index }));
	}, []);

	const setSearchQuery = useCallback((query: string) => {
		setState((prev) => ({ ...prev, searchQuery: query }));
	}, []);

	const showConfirm = useCallback(
		(message: string, onConfirm: () => void, onCancel: () => void) => {
			setState((prev) => ({
				...prev,
				mode: "confirm",
				confirmAction: { message, onConfirm, onCancel },
			}));
		},
		[],
	);

	const hideConfirm = useCallback(() => {
		setState((prev) => ({
			...prev,
			mode: "normal",
			confirmAction: undefined,
		}));
	}, []);

	return {
		state,
		setMode,
		setSelectedIndex,
		setSearchQuery,
		showConfirm,
		hideConfirm,
	};
}
