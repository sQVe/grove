export type ModalMode = "normal" | "filter" | "help" | "confirm";

export interface AppState {
	mode: ModalMode;
	selectedWorktreeIndex: number;
	searchQuery: string;
	showHelp: boolean;
	confirmAction?: {
		message: string;
		onConfirm: () => void;
		onCancel: () => void;
	};
}

export type KeyHandler = (
	input: string,
	key: { name?: string; ctrl?: boolean; meta?: boolean },
) => void;
