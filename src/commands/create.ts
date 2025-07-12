export interface CreateOptions {
	branch: string;
	path?: string;
	checkout?: boolean;
}

export async function createWorktreeCommand({
	branch,
	path,
	checkout = true,
}: CreateOptions) {
	const { createWorktree, isGitRepository, GitOperationError } = await import(
		"../lib/git.js"
	);
	const { resolve } = await import("node:path");

	try {
		// Verify we're in a git repository.
		if (!(await isGitRepository(process.cwd()))) {
			throw new Error(
				"Not in a git repository. Run 'grove init' first or navigate to a git repository.",
			);
		}

		// Create the worktree.
		await createWorktree(branch, path ? resolve(path) : undefined);

		const worktreePath = path ? resolve(path) : resolve("..", branch);

		console.log(
			`✓ Created worktree for branch '${branch}' at: ${worktreePath}`,
		);

		if (checkout) {
			console.log(`✓ Branch '${branch}' checked out`);
		}

		console.log("\nTo switch to this worktree, run:");
		console.log(`  grove switch ${branch}`);
		console.log(`  cd ${worktreePath}`);
	} catch (error) {
		if (error instanceof GitOperationError) {
			if (error.code === "BRANCH_NOT_FOUND") {
				throw new Error(
					`Branch '${branch}' does not exist. Create it first or use an existing branch.`,
				);
			}
			if (error.code === "WORKTREE_EXISTS") {
				throw new Error(
					"Worktree path already exists. Choose a different path or remove the existing worktree.",
				);
			}
			throw new Error(`Failed to create worktree: ${error.message}`);
		}
		throw error;
	}
}
