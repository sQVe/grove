import React from "react";
import { Box, Text } from "ink";
import type { ModalMode } from "./types.js";

interface StatusLineProps {
	mode: ModalMode;
	worktreeCount: number;
}

export function StatusLine({ mode, worktreeCount }: StatusLineProps) {
	const getKeybindings = () => {
		switch (mode) {
			case "filter":
				return "Type to search • [Enter] Apply • [Esc] Cancel";
			case "help":
				return "[?] or [Esc] Close help";
			case "confirm":
				return "[y] Confirm • [n] Cancel";
			default:
				return "[j/k] Navigate • [Enter] Switch • [c] Create • [d] Delete • [/] Filter • [?] Help • [q] Quit";
		}
	};

	const getModeIndicator = () => {
		switch (mode) {
			case "filter":
				return "FILTER";
			case "help":
				return "HELP";
			case "confirm":
				return "CONFIRM";
			default:
				return ""; // No mode indicator for normal browsing
		}
	};

	const modeIndicator = getModeIndicator();

	return (
		<Box borderStyle="single" borderTop paddingX={1}>
			{modeIndicator && (
				<>
					<Text color="cyan" bold>
						{modeIndicator}
					</Text>
					<Text color="gray"> • </Text>
				</>
			)}
			<Text color="gray">{worktreeCount} worktrees</Text>
			<Box flexGrow={1} />
			<Text color="gray">{getKeybindings()}</Text>
		</Box>
	);
}
