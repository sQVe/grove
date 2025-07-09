export interface SwitchOptions {
	worktree: string;
	create?: boolean;
}

export async function switchWorktree({
	worktree,
	create = false,
}: SwitchOptions) {
	throw new Error("switchWorktree not implemented yet");
}
