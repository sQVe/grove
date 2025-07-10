import React from "react";
import { Box, Text } from "ink";
import type { Worktree } from "../commands/list.js";

interface DetailsPanelProps {
	worktree?: Worktree;
}

export function DetailsPanel({ worktree }: DetailsPanelProps) {
	if (!worktree) {
		return (
			<Box flexDirection="column" width="50%" paddingX={1}>
				<Text color="cyan" bold>
					Details
				</Text>
				<Box marginTop={1}>
					<Text color="gray">No worktree selected</Text>
				</Box>
			</Box>
		);
	}

	return (
		<Box flexDirection="column" width="50%" paddingX={1}>
			<Text color="cyan" bold>
				Details
			</Text>
			<Box flexDirection="column" marginTop={1}>
				<Text>
					<Text color="gray">Branch: </Text>
					<Text color="green">{worktree.branch}</Text>
				</Text>
				<Text>
					<Text color="gray">Path: </Text>
					<Text>{worktree.path}</Text>
				</Text>
				<Text>
					<Text color="gray">Status: </Text>
					<Text
						color={
							worktree.active ? "green" : worktree.locked ? "red" : "yellow"
						}
					>
						{worktree.active
							? "✓ Active"
							: worktree.locked
								? "🔒 Locked"
								: "◯ Inactive"}
					</Text>
				</Text>
				<Text>
					<Text color="gray">Head: </Text>
					<Text color="yellow">{worktree.head.slice(0, 8)}</Text>
				</Text>
			</Box>
		</Box>
	);
}
