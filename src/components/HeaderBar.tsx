import React from "react";
import { Box, Text } from "ink";
import type { Worktree } from "../commands/list.js";

interface HeaderBarProps {
	worktrees: Worktree[];
}

export function HeaderBar({ worktrees }: HeaderBarProps) {
	const activeWorktree = worktrees.find((w) => w.active);
	const currentWorktree = activeWorktree?.name || "unknown";
	const worktreeCount = worktrees.length;

	return (
		<Box borderStyle="single" borderBottom paddingX={1}>
			<Text color="cyan" bold>
				Grove
			</Text>
			<Text color="gray"> - </Text>
			<Text color="green">{currentWorktree}</Text>
			<Text color="gray">*</Text>
			<Box flexGrow={1} />
			<Text color="gray">{worktreeCount} worktrees</Text>
		</Box>
	);
}
