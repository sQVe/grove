export interface SwitchOptions {
	worktree: string;
	create?: boolean;
}

export async function switchWorktree({
	worktree,
	create = false,
}: SwitchOptions) {
	const {
		switchToWorktree,
		createWorktree,
		isGitRepository,
		GitOperationError,
	} = await import("../lib/git.js");

	try {
		// Verify we're in a git repository.
		if (!(await isGitRepository(process.cwd()))) {
			throw new Error(
				"Not in a git repository. Run 'grove init' first or navigate to a git repository.",
			);
		}

		try {
			// Try to switch to existing worktree.
			const worktreePath = await switchToWorktree(worktree);

			console.log(`✓ Worktree found: ${worktreePath}`);
			console.log("\nTo switch to this worktree, run:");
			console.log(`  cd ${worktreePath}`);

			// Note: We can't actually change the user's shell directory from Node.js.
			// The user needs to run the cd command themselves.
		} catch (error) {
			if (
				error instanceof GitOperationError &&
				error.code === "WORKTREE_NOT_FOUND"
			) {
				if (create) {
					// Create new worktree.
					console.log(
						`Worktree '${worktree}' not found. Creating new worktree...`,
					);
					await createWorktree(worktree);

					const { resolve } = await import("node:path");
					const worktreePath = resolve("..", worktree);

					console.log(
						`✓ Created new worktree for branch '${worktree}' at: ${worktreePath}`,
					);
					console.log("\nTo switch to this worktree, run:");
					console.log(`  cd ${worktreePath}`);
				} else {
					throw new Error(
						`Worktree '${worktree}' not found. Use --create flag to create a new worktree.`,
					);
				}
			} else {
				throw error;
			}
		}
	} catch (error) {
		if (error instanceof GitOperationError) {
			throw new Error(`Failed to switch worktree: ${error.message}`);
		}
		throw error;
	}
}
