import { exec } from "node:child_process";
import { promisify } from "node:util";
import { access, constants } from "node:fs/promises";
import { join, resolve } from "node:path";
import type { Worktree } from "../commands/list.js";

const execAsync = promisify(exec);

export interface GitError extends Error {
	code?: string;
	stderr?: string;
}

export class GitOperationError extends Error implements GitError {
	constructor(
		message: string,
		public code?: string,
		public stderr?: string,
	) {
		super(message);
		this.name = "GitOperationError";
	}
}

/**
 * Execute a git command with proper error handling.
 */
export async function execGit(
	args: string[],
	cwd = process.cwd(),
): Promise<string> {
	try {
		const { stdout, stderr } = await execAsync(`git ${args.join(" ")}`, {
			cwd,
			encoding: "utf8",
		});

		if (stderr && !stdout) {
			throw new GitOperationError(
				`Git command failed: ${args.join(" ")}`,
				"GIT_ERROR",
				stderr,
			);
		}

		return stdout.trim();
	} catch (error) {
		if (error instanceof Error) {
			throw new GitOperationError(
				`Git command failed: ${args.join(" ")} - ${error.message}`,
				"GIT_EXEC_ERROR",
				error.message,
			);
		}
		throw error;
	}
}

/**
 * Check if a directory is a git repository.
 */
export async function isGitRepository(path: string): Promise<boolean> {
	try {
		await execGit(["rev-parse", "--git-dir"], path);
		return true;
	} catch {
		return false;
	}
}

/**
 * Check if git is available in the system.
 */
export async function isGitAvailable(): Promise<boolean> {
	try {
		await execAsync("git --version");
		return true;
	} catch {
		return false;
	}
}

/**
 * Validate that a path exists and is accessible.
 */
export async function validatePath(path: string): Promise<void> {
	try {
		await access(path, constants.F_OK);
	} catch {
		throw new GitOperationError(
			`Path does not exist or is not accessible: ${path}`,
			"PATH_ERROR",
		);
	}
}

/**
 * Initialize a bare git repository.
 */
export async function initBareRepository(
	path: string,
	defaultBranch = "main",
): Promise<void> {
	const resolvedPath = resolve(path);

	try {
		// Create bare repository.
		await execGit(
			["init", "--bare", "--initial-branch", defaultBranch],
			resolvedPath,
		);
	} catch (error) {
		throw new GitOperationError(
			`Failed to initialize bare repository at ${resolvedPath}`,
			"INIT_ERROR",
			error instanceof GitOperationError ? error.stderr : undefined,
		);
	}
}

/**
 * Parse git worktree list output into Worktree objects.
 */
export function parseWorktreeList(output: string): Worktree[] {
	const worktrees: Worktree[] = [];
	const entries = output.split("\n\n").filter(Boolean);

	for (const entry of entries) {
		const lines = entry.split("\n");
		const worktree: Partial<Worktree> = {};

		for (const line of lines) {
			const [key, ...valueParts] = line.split(" ");
			const value = valueParts.join(" ");

			switch (key) {
				case "worktree":
					worktree.path = value;
					worktree.name = value.split("/").pop() || "unknown";
					break;
				case "HEAD":
					worktree.head = value;
					break;
				case "branch":
					worktree.branch = value.replace("refs/heads/", "");
					break;
				case "bare":
					// Bare repositories don't have a working directory.
					break;
				case "detached":
					worktree.branch = "HEAD";
					break;
			}
		}

		// Determine if worktree is active (current working directory).
		const currentDir = process.cwd();
		worktree.active =
			worktree.path === currentDir ||
			currentDir.startsWith(`${worktree.path}/`);

		// Check if worktree is locked.
		worktree.locked = entry.includes("locked");

		// Only add complete worktree entries.
		if (worktree.path && worktree.name && worktree.head) {
			worktrees.push(worktree as Worktree);
		}
	}

	return worktrees;
}

/**
 * List all worktrees in the repository.
 */
export async function listWorktrees(cwd?: string): Promise<Worktree[]> {
	try {
		const output = await execGit(["worktree", "list", "--porcelain"], cwd);
		return parseWorktreeList(output);
	} catch (error) {
		if (
			error instanceof GitOperationError &&
			error.stderr?.includes("not a git repository")
		) {
			throw new GitOperationError(
				"Not in a git repository or no worktrees found",
				"NOT_GIT_REPO",
				error.stderr,
			);
		}
		throw error;
	}
}

/**
 * Create a new worktree from an existing branch.
 */
export async function createWorktree(
	branch: string,
	path?: string,
	cwd?: string,
): Promise<void> {
	const worktreePath = path ?? join(process.cwd(), "..", branch);

	try {
		// Check if branch exists.
		await execGit(["show-ref", "--verify", `refs/heads/${branch}`], cwd);

		// Create worktree.
		await execGit(["worktree", "add", worktreePath, branch], cwd);
	} catch (error) {
		if (
			error instanceof GitOperationError &&
			error.stderr?.includes("already exists")
		) {
			throw new GitOperationError(
				`Worktree path already exists: ${worktreePath}`,
				"WORKTREE_EXISTS",
				error.stderr,
			);
		}
		if (
			error instanceof GitOperationError &&
			error.stderr?.includes("invalid reference")
		) {
			throw new GitOperationError(
				`Branch does not exist: ${branch}`,
				"BRANCH_NOT_FOUND",
				error.stderr,
			);
		}
		throw error;
	}
}

/**
 * Remove a worktree.
 */
export async function removeWorktree(
	path: string,
	cwd?: string,
): Promise<void> {
	try {
		await execGit(["worktree", "remove", path], cwd);
	} catch (error) {
		if (
			error instanceof GitOperationError &&
			error.stderr?.includes("not a working tree")
		) {
			throw new GitOperationError(
				`Path is not a worktree: ${path}`,
				"NOT_WORKTREE",
				error.stderr,
			);
		}
		throw error;
	}
}

/**
 * Switch to a worktree by changing the current working directory.
 * Note: This only validates the worktree exists - actual directory change
 * must be handled by the calling process.
 */
export async function switchToWorktree(
	worktreeName: string,
	cwd?: string,
): Promise<string> {
	const worktrees = await listWorktrees(cwd);
	const targetWorktree = worktrees.find(
		(w) => w.name === worktreeName || w.path.endsWith(worktreeName),
	);

	if (!targetWorktree) {
		throw new GitOperationError(
			`Worktree not found: ${worktreeName}`,
			"WORKTREE_NOT_FOUND",
		);
	}

	// Validate the worktree path exists.
	await validatePath(targetWorktree.path);

	return targetWorktree.path;
}
