export interface CloneOptions {
	branch: string;
	path?: string;
	checkout?: boolean;
}

export async function cloneWorktree({
	branch,
	path,
	checkout = true,
}: CloneOptions) {
	throw new Error("cloneWorktree not implemented yet");
}
