export interface Worktree {
	name: string;
	path: string;
	branch: string;
	head: string;
	active: boolean;
	locked: boolean;
}

export interface ListOptions {
	format?: "table" | "json";
	showLocked?: boolean;
}

export async function listWorktrees({
	format = "table",
	showLocked = false,
}: ListOptions = {}): Promise<Worktree[]> {
	throw new Error("listWorktrees not implemented yet");
}
