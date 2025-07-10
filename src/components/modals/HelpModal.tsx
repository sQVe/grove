import React from "react";
import { Box, Text } from "ink";

export function HelpModal() {
	return (
		<Box
			borderStyle="double"
			borderColor="cyan"
			paddingX={2}
			paddingY={1}
			marginX={4}
			marginTop={1}
		>
			<Box flexDirection="column">
				<Text color="cyan" bold>
					Grove - Keyboard Shortcuts
				</Text>

				<Box marginTop={1} flexDirection="column">
					<Text color="yellow" bold>
						Navigation:
					</Text>
					<Text>
						{" "}
						<Text color="cyan">j / ↓</Text> Move down
					</Text>
					<Text>
						{" "}
						<Text color="cyan">k / ↑</Text> Move up
					</Text>
					<Text>
						{" "}
						<Text color="cyan">Enter</Text> Switch to worktree
					</Text>
				</Box>

				<Box marginTop={1} flexDirection="column">
					<Text color="yellow" bold>
						Actions:
					</Text>
					<Text>
						{" "}
						<Text color="cyan">c</Text> Create new worktree
					</Text>
					<Text>
						{" "}
						<Text color="cyan">d</Text> Delete worktree
					</Text>
					<Text>
						{" "}
						<Text color="cyan">r</Text> Rename worktree
					</Text>
				</Box>

				<Box marginTop={1} flexDirection="column">
					<Text color="yellow" bold>
						Search & Help:
					</Text>
					<Text>
						{" "}
						<Text color="cyan">/</Text> Filter worktrees
					</Text>
					<Text>
						{" "}
						<Text color="cyan">?</Text> Toggle this help
					</Text>
				</Box>

				<Box marginTop={1} flexDirection="column">
					<Text color="yellow" bold>
						Exit:
					</Text>
					<Text>
						{" "}
						<Text color="cyan">q</Text> Quit Grove
					</Text>
					<Text>
						{" "}
						<Text color="cyan">Esc</Text> Cancel current action
					</Text>
				</Box>

				<Box marginTop={2}>
					<Text color="gray">
						Press <Text color="cyan">?</Text> or <Text color="cyan">Esc</Text>{" "}
						to close this help
					</Text>
				</Box>
			</Box>
		</Box>
	);
}
