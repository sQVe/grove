import { exec } from "node:child_process";
import { promisify } from "node:util";
import { access, constants } from "node:fs/promises";
import { join, resolve } from "node:path";
import type { Worktree } from "../commands/list.js";

const execAsync = promisify(exec);

/**
 * Enhanced execution options for git commands.
 */
export interface GitExecOptions {
	cwd?: string;
	maxBuffer?: number;
	timeout?: number;
	env?: Record<string, string>;
}

/**
 * Git command execution result with enhanced context.
 */
export interface GitExecResult {
	stdout: string;
	stderr: string;
	command: string;
	cwd: string;
	exitCode?: number;
}

export interface GitError extends Error {
	code?: string;
	stderr?: string;
	command?: string;
	cwd?: string;
	exitCode?: number;
}

export class GitOperationError extends Error implements GitError {
	constructor(
		message: string,
		public code?: string,
		public stderr?: string,
		public command?: string,
		public cwd?: string,
		public exitCode?: number,
	) {
		super(message);
		this.name = "GitOperationError";
	}

	/**
	 * Create a GitOperationError from an execution result.
	 */
	static fromExecResult(
		message: string,
		result: GitExecResult,
		code?: string,
	): GitOperationError {
		return new GitOperationError(
			message,
			code,
			result.stderr,
			result.command,
			result.cwd,
			result.exitCode,
		);
	}

	/**
	 * Create a GitOperationError from an existing error with additional context.
	 */
	static fromError(
		message: string,
		error: GitOperationError,
		code?: string,
	): GitOperationError {
		return new GitOperationError(
			message,
			code || error.code,
			error.stderr,
			error.command,
			error.cwd,
			error.exitCode,
		);
	}
}

/**
 * Execute a command with enhanced PATH environment to ensure git is available.
 * Based on Crystal's execWithShellPath pattern.
 */
export async function execWithShellPath(
	command: string,
	options: GitExecOptions = {},
): Promise<GitExecResult> {
	const cwd = options.cwd ?? process.cwd();
	const maxBuffer = options.maxBuffer ?? 1024 * 1024 * 10; // 10MB default
	const timeout = options.timeout ?? 30000; // 30s default

	// Ensure git is in PATH by adding common git installation paths
	const isWindows = process.platform === "win32";
	const pathSeparator = isWindows ? ";" : ":";
	const commonGitPaths = isWindows
		? ["C:\\Program Files\\Git\\bin", "C:\\Program Files (x86)\\Git\\bin"]
		: ["/usr/bin", "/usr/local/bin", "/opt/homebrew/bin", "/usr/local/git/bin"];

	const currentPath = process.env.PATH || "";
	const enhancedPath = [currentPath, ...commonGitPaths].join(pathSeparator);

	const env = {
		...process.env,
		...options.env,
		PATH: enhancedPath,
	};

	try {
		const { stdout, stderr } = await execAsync(command, {
			cwd,
			encoding: "utf8",
			maxBuffer,
			timeout,
			env,
		});

		return {
			stdout: stdout.trim(),
			stderr: stderr.trim(),
			command,
			cwd,
		};
	} catch (error: unknown) {
		const execError = error as {
			code?: number;
			stderr?: string;
			stdout?: string;
			message?: string;
		};
		const exitCode = execError.code;
		const stderr = execError.stderr || execError.message || "";
		const stdout = execError.stdout || "";

		return {
			stdout: stdout.trim(),
			stderr: stderr.trim(),
			command,
			cwd,
			exitCode,
		};
	}
}

/**
 * Execute a git command with enhanced error handling and context.
 * Inspired by Crystal's robust git command execution patterns.
 */
export async function execGit(
	args: string[],
	options: GitExecOptions = {},
): Promise<string> {
	const command = `git ${args.join(" ")}`;
	const result = await execWithShellPath(command, options);

	// Handle git command failures
	if (result.exitCode && result.exitCode !== 0) {
		// Check for common git errors and provide better error messages
		if (result.stderr.includes("not a git repository")) {
			throw GitOperationError.fromExecResult(
				"Not in a git repository",
				result,
				"NOT_GIT_REPO",
			);
		}

		if (result.stderr.includes("does not exist")) {
			throw GitOperationError.fromExecResult(
				"Git reference does not exist",
				result,
				"REF_NOT_FOUND",
			);
		}

		if (result.stderr.includes("already exists")) {
			throw GitOperationError.fromExecResult(
				"Resource already exists",
				result,
				"ALREADY_EXISTS",
			);
		}

		if (result.stderr.includes("Permission denied")) {
			throw GitOperationError.fromExecResult(
				"Permission denied",
				result,
				"PERMISSION_DENIED",
			);
		}

		throw GitOperationError.fromExecResult(
			`Git command failed: ${args.join(" ")}`,
			result,
			"GIT_ERROR",
		);
	}

	// Some git commands write informational output to stderr but still succeed
	if (result.stderr && !result.stdout && result.exitCode === undefined) {
		// Only treat as error if stderr contains actual error indicators
		if (
			result.stderr.includes("error:") ||
			result.stderr.includes("fatal:") ||
			result.stderr.includes("warning:")
		) {
			throw GitOperationError.fromExecResult(
				`Git command failed: ${args.join(" ")}`,
				result,
				"GIT_ERROR",
			);
		}
	}

	return result.stdout;
}

/**
 * Check if a directory is a git repository.
 */
export async function isGitRepository(path: string): Promise<boolean> {
	try {
		await execGit(["rev-parse", "--git-dir"], { cwd: path });
		return true;
	} catch {
		return false;
	}
}

/**
 * Check if git is available in the system using enhanced PATH.
 */
export async function isGitAvailable(): Promise<boolean> {
	try {
		const result = await execWithShellPath("git --version");
		return result.exitCode === undefined || result.exitCode === 0;
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
 * Initialize a bare git repository with enhanced error handling.
 */
export async function initBareRepository(
	path: string,
	defaultBranch = "main",
): Promise<void> {
	const resolvedPath = resolve(path);

	try {
		// Create bare repository.
		await execGit(["init", "--bare", "--initial-branch", defaultBranch], {
			cwd: resolvedPath,
		});
	} catch (error) {
		if (error instanceof GitOperationError) {
			throw GitOperationError.fromError(
				`Failed to initialize bare repository at ${resolvedPath}: ${error.message}`,
				error,
				"INIT_ERROR",
			);
		}
		throw new GitOperationError(
			`Failed to initialize bare repository at ${resolvedPath}`,
			"INIT_ERROR",
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
 * List all worktrees in the repository with enhanced error handling.
 */
export async function listWorktrees(cwd?: string): Promise<Worktree[]> {
	try {
		const output = await execGit(["worktree", "list", "--porcelain"], { cwd });
		return parseWorktreeList(output);
	} catch (error) {
		if (error instanceof GitOperationError && error.code === "NOT_GIT_REPO") {
			throw GitOperationError.fromError(
				"Not in a git repository or no worktrees found",
				error,
				"NOT_GIT_REPO",
			);
		}
		throw error;
	}
}

/**
 * Create a new worktree from an existing branch with enhanced error handling.
 */
export async function createWorktree(
	branch: string,
	path?: string,
	cwd?: string,
): Promise<void> {
	const worktreePath = path ?? join(process.cwd(), "..", branch);

	try {
		// Check if branch exists.
		await execGit(["show-ref", "--verify", `refs/heads/${branch}`], { cwd });

		// Create worktree.
		await execGit(["worktree", "add", worktreePath, branch], { cwd });
	} catch (error) {
		if (error instanceof GitOperationError && error.code === "ALREADY_EXISTS") {
			throw GitOperationError.fromError(
				`Worktree path already exists: ${worktreePath}`,
				error,
				"WORKTREE_EXISTS",
			);
		}
		if (error instanceof GitOperationError && error.code === "REF_NOT_FOUND") {
			throw GitOperationError.fromError(
				`Branch does not exist: ${branch}`,
				error,
				"BRANCH_NOT_FOUND",
			);
		}
		throw error;
	}
}

/**
 * Remove a worktree with enhanced error handling.
 */
export async function removeWorktree(
	path: string,
	cwd?: string,
): Promise<void> {
	try {
		await execGit(["worktree", "remove", path], { cwd });
	} catch (error) {
		if (
			error instanceof GitOperationError &&
			error.stderr?.includes("not a working tree")
		) {
			throw GitOperationError.fromError(
				`Path is not a worktree: ${path}`,
				error,
				"NOT_WORKTREE",
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

/**
 * Get the current branch name or HEAD status.
 * Handles detached HEAD and edge cases.
 */
export async function getCurrentBranch(cwd?: string): Promise<string> {
	try {
		// Try to get current branch name
		const branch = await execGit(["branch", "--show-current"], { cwd });
		if (branch) {
			return branch;
		}

		// Fallback for detached HEAD
		const head = await execGit(["rev-parse", "--short", "HEAD"], { cwd });
		return `HEAD@${head}`;
	} catch (error) {
		if (error instanceof GitOperationError && error.code === "NOT_GIT_REPO") {
			throw error;
		}
		// Repository might have no commits yet
		return "main";
	}
}

/**
 * Check if repository has any commits.
 */
export async function hasCommits(cwd?: string): Promise<boolean> {
	try {
		await execGit(["rev-parse", "HEAD"], { cwd });
		return true;
	} catch {
		return false;
	}
}

/**
 * Get repository status information.
 * Returns file change statistics and status.
 */
export interface RepositoryStatus {
	clean: boolean;
	modified: string[];
	added: string[];
	deleted: string[];
	untracked: string[];
	ahead: number;
	behind: number;
}

/**
 * Get detailed repository status using git status --porcelain.
 * Inspired by Crystal's file tracking patterns.
 */
export async function getRepositoryStatus(
	cwd?: string,
): Promise<RepositoryStatus> {
	try {
		// Get working directory status
		const statusOutput = await execGit(["status", "--porcelain"], { cwd });

		const modified: string[] = [];
		const added: string[] = [];
		const deleted: string[] = [];
		const untracked: string[] = [];

		// Parse porcelain output
		for (const line of statusOutput.split("\n").filter(Boolean)) {
			const status = line.substring(0, 2);
			const file = line.substring(3);

			if (status.includes("M")) {
				modified.push(file);
			} else if (status.includes("A")) {
				added.push(file);
			} else if (status.includes("D")) {
				deleted.push(file);
			} else if (status.includes("??")) {
				untracked.push(file);
			}
		}

		// Get ahead/behind status
		let ahead = 0;
		let behind = 0;

		try {
			const branch = await getCurrentBranch(cwd);
			if (branch && !branch.startsWith("HEAD@")) {
				const aheadBehind = await execGit(
					["rev-list", "--left-right", "--count", `origin/${branch}...HEAD`],
					{ cwd },
				);
				const [behindStr, aheadStr] = aheadBehind.split("\t");
				behind = Number.parseInt(behindStr || "0", 10);
				ahead = Number.parseInt(aheadStr || "0", 10);
			}
		} catch {
			// Ignore errors for ahead/behind calculation
		}

		return {
			clean:
				modified.length === 0 &&
				added.length === 0 &&
				deleted.length === 0 &&
				untracked.length === 0,
			modified,
			added,
			deleted,
			untracked,
			ahead,
			behind,
		};
	} catch (error) {
		if (error instanceof GitOperationError && error.code === "NOT_GIT_REPO") {
			throw error;
		}
		// Return empty status for other errors
		return {
			clean: true,
			modified: [],
			added: [],
			deleted: [],
			untracked: [],
			ahead: 0,
			behind: 0,
		};
	}
}

/**
 * Initialize a new git repository with first commit.
 * Handles the edge case of repositories with no commits.
 */
export async function initRepository(
	path: string,
	defaultBranch = "main",
): Promise<void> {
	const resolvedPath = resolve(path);

	try {
		// Initialize repository
		await execGit(["init", "--initial-branch", defaultBranch], {
			cwd: resolvedPath,
		});

		// Create initial commit if no commits exist
		const hasInitialCommits = await hasCommits(resolvedPath);
		if (!hasInitialCommits) {
			// Create .gitkeep to ensure we can make an initial commit
			await execGit(["config", "user.name", "Grove"], { cwd: resolvedPath });
			await execGit(["config", "user.email", "grove@example.com"], {
				cwd: resolvedPath,
			});
			await execGit(["commit", "--allow-empty", "-m", "Initial commit"], {
				cwd: resolvedPath,
			});
		}
	} catch (error) {
		if (error instanceof GitOperationError) {
			throw GitOperationError.fromError(
				`Failed to initialize repository at ${resolvedPath}: ${error.message}`,
				error,
				"INIT_ERROR",
			);
		}
		throw new GitOperationError(
			`Failed to initialize repository at ${resolvedPath}`,
			"INIT_ERROR",
		);
	}
}

/**
 * Create a new branch from an existing branch or commit.
 * Handles edge cases and provides flexible configuration.
 */
export async function createBranch(
	branchName: string,
	fromBranch?: string,
	cwd?: string,
): Promise<void> {
	try {
		const args = ["checkout", "-b", branchName];
		if (fromBranch) {
			args.push(fromBranch);
		}

		await execGit(args, { cwd });
	} catch (error) {
		if (error instanceof GitOperationError && error.code === "ALREADY_EXISTS") {
			throw GitOperationError.fromError(
				`Branch already exists: ${branchName}`,
				error,
				"BRANCH_EXISTS",
			);
		}
		throw error;
	}
}

/**
 * Enhanced worktree creation with flexible configuration.
 * Inspired by Crystal's dynamic worktree creation patterns.
 */
export async function createWorktreeAdvanced({
	branch,
	path,
	fromBranch,
	createNewBranch = false,
	cwd,
}: {
	branch: string;
	path?: string;
	fromBranch?: string;
	createNewBranch?: boolean;
	cwd?: string;
}): Promise<void> {
	const worktreePath = path ?? join(process.cwd(), "..", branch);

	if (createNewBranch) {
		// Create new branch and worktree in one command
		const args = ["worktree", "add", "-b", branch, worktreePath];
		if (fromBranch) {
			args.push(fromBranch);
		}
		await execGit(args, { cwd });
	} else {
		// Create worktree from existing branch
		await createWorktree(branch, path, cwd);
	}
}
