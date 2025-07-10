import React from "react";
import { Box, Text } from "ink";
import type { Worktree } from "../commands/list.js";
import type { ModalMode } from "./types.js";

interface WorktreeListPanelProps {
	worktrees: Worktree[];
	selectedIndex: number;
	mode: ModalMode;
}

export function WorktreeListPanel({
	worktrees,
	selectedIndex,
	mode,
}: WorktreeListPanelProps) {
	return (
		<Box
			flexDirection="column"
			width="50%"
			borderStyle="single"
			borderRight
			paddingX={1}
		>
			<Text color="cyan" bold>
				Worktrees
			</Text>
			<Box flexDirection="column" marginTop={1}>
				{worktrees.length === 0 ? (
					<Text color="gray">No worktrees found</Text>
				) : (
					worktrees.map((worktree, index) => {
						const isSelected = index === selectedIndex && mode === "normal";
						const status = worktree.active
							? "*active"
							: worktree.locked
								? "locked"
								: "";

						return (
							<Box key={worktree.path}>
								<Text
									color={isSelected ? "black" : undefined}
									backgroundColor={isSelected ? "cyan" : undefined}
								>
									{isSelected ? "> " : "  "}
									{worktree.name.padEnd(20)}
									<Text color="gray">{status}</Text>
								</Text>
							</Box>
						);
					})
				)}
			</Box>
		</Box>
	);
}
