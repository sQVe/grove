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
	const { listWorktrees: gitListWorktrees, GitOperationError } = await import(
		"../lib/git.js"
	);

	try {
		const worktrees = await gitListWorktrees();

		// Filter out locked worktrees unless explicitly requested.
		const filteredWorktrees = showLocked
			? worktrees
			: worktrees.filter((w) => !w.locked);

		// Output based on format.
		if (format === "json") {
			console.log(JSON.stringify(filteredWorktrees, null, 2));
		} else {
			// Table format.
			if (filteredWorktrees.length === 0) {
				console.log("No worktrees found.");
				return [];
			}

			console.log("Worktrees:");
			console.log(
				`${"Name".padEnd(20)}${"Branch".padEnd(20)}${"Path".padEnd(40)}Status`,
			);
			console.log("-".repeat(80));

			for (const worktree of filteredWorktrees) {
				const status = worktree.active
					? "*active"
					: worktree.locked
						? "locked"
						: "";
				console.log(
					worktree.name.padEnd(20) +
						(worktree.branch || "").padEnd(20) +
						worktree.path.padEnd(40) +
						status,
				);
			}
		}

		return filteredWorktrees;
	} catch (error) {
		if (error instanceof GitOperationError) {
			throw new Error(`Failed to list worktrees: ${error.message}`);
		}
		throw error;
	}
}
