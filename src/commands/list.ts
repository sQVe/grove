import chalk from "chalk";
import figures from "figures";

export interface Worktree {
	name: string;
	path: string;
	branch: string;
	head: string;
	active: boolean;
	locked: boolean;
}

export interface WorktreeWithStatus extends Worktree {
	isDirty?: boolean;
	ahead?: number;
	behind?: number;
}

export interface ListOptions {
	format?: "table" | "json";
	showLocked?: boolean;
	showStatus?: boolean;
}

export async function listWorktrees({
	format = "table",
	showLocked = false,
	showStatus = true,
}: ListOptions = {}): Promise<Worktree[]> {
	const {
		listWorktrees: gitListWorktrees,
		GitOperationError,
		getRepositoryStatus,
	} = await import("../lib/git.js");

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
				console.log(chalk.yellow("No worktrees found."));
				return [];
			}

			await displayWorktreesTable(filteredWorktrees, showStatus);
		}

		return filteredWorktrees;
	} catch (error) {
		if (error instanceof GitOperationError) {
			throw new Error(`Failed to list worktrees: ${error.message}`);
		}
		throw error;
	}
}

async function displayWorktreesTable(
	worktrees: Worktree[],
	showStatus: boolean,
): Promise<void> {
	const { getRepositoryStatus } = await import("../lib/git.js");

	// Enhance worktrees with status information if requested.
	const enhancedWorktrees: WorktreeWithStatus[] = [];

	for (const worktree of worktrees) {
		const enhanced: WorktreeWithStatus = { ...worktree };

		if (showStatus) {
			try {
				const status = await getRepositoryStatus(worktree.path);
				enhanced.isDirty = !status.clean;
				enhanced.ahead = status.ahead;
				enhanced.behind = status.behind;
			} catch {
				// Ignore status errors for individual worktrees.
			}
		}

		enhancedWorktrees.push(enhanced);
	}

	// Display the header.
	console.log(chalk.bold.blue("Worktrees:"));
	console.log();

	// Calculate column widths.
	const nameWidth = Math.max(
		...enhancedWorktrees.map((w) => w.name.length),
		"Name".length,
	);
	const branchWidth = Math.max(
		...enhancedWorktrees.map((w) => (w.branch || "").length),
		"Branch".length,
	);
	const pathWidth = Math.max(
		...enhancedWorktrees.map((w) => w.path.length),
		"Path".length,
		40,
	);

	// Display column headers.
	const headers = [
		chalk.bold(""),
		chalk.bold("Name".padEnd(nameWidth + 2)),
		chalk.bold("Branch".padEnd(branchWidth + 2)),
		chalk.bold("Path".padEnd(pathWidth + 2)),
		chalk.bold("Status"),
	];
	console.log(headers.join(""));

	// Display separator.
	const separatorLength =
		2 + nameWidth + 2 + branchWidth + 2 + pathWidth + 2 + 10;
	console.log(chalk.dim("─".repeat(separatorLength)));

	// Display each worktree.
	for (const worktree of enhancedWorktrees) {
		const statusIcon = getStatusIcon(worktree);
		const name = formatName(worktree, nameWidth);
		const branch = formatBranch(worktree, branchWidth);
		const path = formatPath(worktree, pathWidth);
		const status = formatStatus(worktree);

		console.log(`${statusIcon} ${name} ${branch} ${path} ${status}`);
	}

	console.log();
}

function getStatusIcon(worktree: WorktreeWithStatus): string {
	if (worktree.active) {
		return chalk.green(figures.pointer);
	}
	if (worktree.locked) {
		return chalk.red(figures.cross);
	}
	if (worktree.isDirty) {
		return chalk.yellow(figures.bullet);
	}
	return chalk.dim(figures.circleDotted);
}

function formatName(worktree: WorktreeWithStatus, width: number): string {
	const name = worktree.name.padEnd(width + 2);
	if (worktree.active) {
		return chalk.green.bold(name);
	}
	if (worktree.locked) {
		return chalk.red(name);
	}
	return chalk.white(name);
}

function formatBranch(worktree: WorktreeWithStatus, width: number): string {
	const branch = (worktree.branch || "").padEnd(width + 2);
	if (worktree.branch === "HEAD") {
		return chalk.yellow(branch);
	}
	if (worktree.active) {
		return chalk.green(branch);
	}
	return chalk.cyan(branch);
}

function formatPath(worktree: WorktreeWithStatus, width: number): string {
	const path = worktree.path.padEnd(width + 2);
	if (worktree.active) {
		return chalk.green(path);
	}
	return chalk.dim(path);
}

function formatStatus(worktree: WorktreeWithStatus): string {
	const statusParts: string[] = [];

	if (worktree.active) {
		statusParts.push(chalk.green.bold("active"));
	}

	if (worktree.locked) {
		statusParts.push(chalk.red("locked"));
	}

	if (worktree.isDirty) {
		statusParts.push(chalk.yellow("dirty"));
	}

	if (worktree.ahead && worktree.ahead > 0) {
		statusParts.push(chalk.green(`${figures.arrowUp}${worktree.ahead}`));
	}

	if (worktree.behind && worktree.behind > 0) {
		statusParts.push(chalk.red(`${figures.arrowDown}${worktree.behind}`));
	}

	if (statusParts.length === 0) {
		statusParts.push(chalk.green("clean"));
	}

	return statusParts.join(" ");
}
