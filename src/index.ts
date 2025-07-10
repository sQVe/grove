#!/usr/bin/env node

import yargs from "yargs";
import { hideBin } from "yargs/helpers";
import chalk from "chalk";
import figures from "figures";

async function main() {
	const cli = yargs(hideBin(process.argv))
		.scriptName("grove")
		.usage("$0 <command>")
		.example("$0", "Launch interactive TUI")
		.example("$0 init", "Initialize bare repository")
		.example("$0 clone feature-branch", "Clone worktree for branch")
		.example("$0 switch main", "Switch to main worktree")
		.example("$0 list", "List all worktrees")
		.command(
			"init [path]",
			"Initialize bare repository with grove configuration",
			(yargs) => {
				return yargs
					.positional("path", {
						describe: "Path to initialize repository",
						type: "string",
						default: process.cwd(),
					})
					.option("bare", {
						describe: "Initialize as bare repository",
						type: "boolean",
						default: true,
					})
					.option("default-branch", {
						describe: "Default branch name",
						type: "string",
						default: "main",
					});
			},
			async (argv) => {
				await handleInit(argv.path, {
					bare: argv.bare,
					defaultBranch: argv.defaultBranch,
				});
			},
		)
		.command(
			"clone <branch> [path]",
			"Clone worktree from existing branch",
			(yargs) => {
				return yargs
					.positional("branch", {
						describe: "Branch name to clone",
						type: "string",
						demandOption: true,
					})
					.positional("path", {
						describe: "Path for the new worktree",
						type: "string",
					})
					.option("checkout", {
						describe: "Checkout branch after cloning",
						type: "boolean",
						default: true,
					});
			},
			async (argv) => {
				await handleClone(argv.branch, argv.path, { checkout: argv.checkout });
			},
		)
		.command(
			"switch <worktree>",
			"Switch to specified worktree",
			(yargs) => {
				return yargs
					.positional("worktree", {
						describe: "Worktree name to switch to",
						type: "string",
						demandOption: true,
					})
					.option("create", {
						describe: "Create worktree if it doesn't exist",
						type: "boolean",
						default: false,
					});
			},
			async (argv) => {
				await handleSwitch(argv.worktree, { create: argv.create });
			},
		)
		.command(
			"list",
			"List all worktrees with status",
			(yargs) => {
				return yargs
					.option("format", {
						describe: "Output format",
						type: "string",
						choices: ["table", "json"],
						default: "table",
					})
					.option("show-locked", {
						describe: "Show locked worktrees",
						type: "boolean",
						default: false,
					});
			},
			async (argv) => {
				await handleList({
					format: argv.format as "table" | "json",
					showLocked: argv.showLocked,
				});
			},
		)
		.help()
		.version()
		.completion("completion", "Generate shell completion script")
		.demandCommand(0, 1, "", "Too many commands specified")
		.strict();

	const argv = await cli.parseAsync();

	if (argv._.length === 0) {
		await launchTUI();
	}
}

async function handleInit(
	path: string,
	options: { bare: boolean; defaultBranch: string },
) {
	const { initRepository } = await import("./commands/init.js");

	try {
		await initRepository({
			path,
			bare: options.bare,
			defaultBranch: options.defaultBranch,
		});
	} catch (error) {
		console.error(
			chalk.red(figures.cross),
			"Error:",
			error instanceof Error ? error.message : error,
		);
		process.exit(1);
	}
}

async function handleClone(
	branch: string,
	path?: string,
	options: { checkout: boolean } = { checkout: true },
) {
	const { cloneWorktree } = await import("./commands/clone.js");

	try {
		await cloneWorktree({
			branch,
			path,
			checkout: options.checkout,
		});
	} catch (error) {
		console.error(
			chalk.red(figures.cross),
			"Error:",
			error instanceof Error ? error.message : error,
		);
		process.exit(1);
	}
}

async function handleSwitch(
	worktree: string,
	options: { create: boolean } = { create: false },
) {
	const { switchWorktree } = await import("./commands/switch.js");

	try {
		await switchWorktree({
			worktree,
			create: options.create,
		});
	} catch (error) {
		console.error(
			chalk.red(figures.cross),
			"Error:",
			error instanceof Error ? error.message : error,
		);
		process.exit(1);
	}
}

async function handleList(
	options: { format: "table" | "json"; showLocked: boolean } = {
		format: "table",
		showLocked: false,
	},
) {
	const { listWorktrees } = await import("./commands/list.js");

	try {
		await listWorktrees({
			format: options.format,
			showLocked: options.showLocked,
		});
	} catch (error) {
		console.error(
			chalk.red(figures.cross),
			"Error:",
			error instanceof Error ? error.message : error,
		);
		process.exit(1);
	}
}

async function launchTUI() {
	const { launchTUI: startTUI } = await import("./components/tui");
	await startTUI();
}

main().catch((error) => {
	console.error(chalk.red(figures.cross), "Error:", error.message);
	process.exit(1);
});
