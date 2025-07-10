import React from "react";
import { Box, Text, useInput } from "ink";
import type { Worktree } from "../../commands/list.js";
import { searchWorktrees } from "../../lib/fuzzy.js";

interface FilterModalProps {
	query: string;
	onQueryChange: (query: string) => void;
	onApply: () => void;
	onCancel: () => void;
	allWorktrees: Worktree[];
}

export function FilterModal({
	query,
	onQueryChange,
	onApply,
	onCancel,
	allWorktrees,
}: FilterModalProps) {
	useInput((input, key) => {
		if (key.return) {
			onApply();
		} else if (key.escape) {
			onCancel();
		} else if (key.backspace) {
			onQueryChange(query.slice(0, -1));
		} else if (input && !key.ctrl && !key.meta) {
			onQueryChange(query + input);
		}
	});

	return (
		<Box
			borderStyle="double"
			borderColor="cyan"
			paddingX={1}
			paddingY={1}
			marginX={2}
			marginTop={2}
		>
			<Box flexDirection="column">
				<Text color="cyan" bold>
					Filter Worktrees
				</Text>
				<Box marginTop={1}>
					<Text color="gray">Search: </Text>
					<Text color="white">{query}</Text>
					<Text color="cyan">█</Text>
				</Box>
			</Box>
		</Box>
	);
}
