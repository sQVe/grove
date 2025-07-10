import React from "react";
import { Box, Text, useInput } from "ink";

interface ConfirmModalProps {
	message: string;
	onConfirm: () => void;
	onCancel: () => void;
}

export function ConfirmModal({
	message,
	onConfirm,
	onCancel,
}: ConfirmModalProps) {
	useInput((input, key) => {
		if (input === "y" || key.return) {
			onConfirm();
		} else if (input === "n" || key.escape) {
			onCancel();
		}
	});

	return (
		<Box
			borderStyle="double"
			borderColor="yellow"
			paddingX={2}
			paddingY={1}
			marginX={4}
			marginTop={3}
		>
			<Box flexDirection="column" alignItems="center">
				<Text color="yellow" bold>
					Confirm Action
				</Text>
				<Box marginTop={1} marginBottom={1}>
					<Text>{message}</Text>
				</Box>
				<Box>
					<Text color="gray">
						[<Text color="green">y</Text>] Yes [<Text color="red">n</Text>] No
					</Text>
				</Box>
			</Box>
		</Box>
	);
}
