import React, { useEffect, useState } from "react";
import { Box, useInput } from "ink";
import type { Worktree } from "../commands/list.js";
import { searchWorktrees } from "../lib/fuzzy.js";
import { useModalState } from "./hooks/useModalState";
import { HeaderBar } from "./HeaderBar";
import { WorktreeListPanel } from "./WorktreeListPanel";
import { DetailsPanel } from "./DetailsPanel";
import { StatusLine } from "./StatusLine";
import { FilterModal } from "./modals/FilterModal";
import { HelpModal } from "./modals/HelpModal";
import { ConfirmModal } from "./modals/ConfirmModal";

interface AppProps {
	initialWorktrees?: Worktree[];
	onExit?: () => void;
}

export function App({ initialWorktrees = [], onExit }: AppProps) {
	const [worktrees, setWorktrees] = useState<Worktree[]>(initialWorktrees);
	const [filteredWorktrees, setFilteredWorktrees] =
		useState<Worktree[]>(initialWorktrees);
	const modalState = useModalState();

	// Load worktrees on mount.
	useEffect(() => {
		async function loadWorktrees() {
			try {
				const { listWorktrees } = await import("../lib/git.js");
				const loadedWorktrees = await listWorktrees();
				setWorktrees(loadedWorktrees);
				setFilteredWorktrees(loadedWorktrees);
			} catch (error) {
				// Handle error - for now just use empty array.
				setWorktrees([]);
				setFilteredWorktrees([]);
			}
		}

		if (initialWorktrees.length === 0) {
			loadWorktrees();
		}
	}, [initialWorktrees.length]);

	// Apply fuzzy search when search query changes.
	useEffect(() => {
		const query = modalState.state.searchQuery;
		if (query.trim() === "") {
			setFilteredWorktrees(worktrees);
		} else {
			const results = searchWorktrees(worktrees, query);
			setFilteredWorktrees(results);
		}
		// Reset selection when results change.
		modalState.setSelectedIndex(0);
	}, [worktrees, modalState.state.searchQuery, modalState.setSelectedIndex]);

	// Navigation helpers
	const handleNavigateDown = () => {
		if (filteredWorktrees.length === 0) return;

		const currentIndex = modalState.state.selectedWorktreeIndex;
		const nextIndex =
			currentIndex >= filteredWorktrees.length - 1
				? 0 // Wrap to beginning
				: currentIndex + 1;
		modalState.setSelectedIndex(nextIndex);
	};

	const handleNavigateUp = () => {
		if (filteredWorktrees.length === 0) return;

		const currentIndex = modalState.state.selectedWorktreeIndex;
		const prevIndex =
			currentIndex <= 0
				? filteredWorktrees.length - 1 // Wrap to end
				: currentIndex - 1;
		modalState.setSelectedIndex(prevIndex);
	};

	// Action handlers
	const handleSwitchWorktree = async (worktree: Worktree) => {
		try {
			const { switchToWorktree } = await import("../lib/git.js");
			await switchToWorktree(worktree.path);
			// Refresh worktrees after switching
			const { listWorktrees } = await import("../lib/git.js");
			const updatedWorktrees = await listWorktrees();
			setWorktrees(updatedWorktrees);
			setFilteredWorktrees(updatedWorktrees);
		} catch (error) {
			// TODO: Show error message to user
			console.error("Failed to switch worktree:", error);
		}
	};

	const handleCreateWorktree = () => {
		// TODO: Implement create worktree modal/prompt
		modalState.showConfirm(
			"Create worktree feature not implemented yet",
			() => modalState.hideConfirm(),
			() => modalState.hideConfirm(),
		);
	};

	const handleDeleteWorktree = (worktree: Worktree) => {
		modalState.showConfirm(
			`Delete worktree "${worktree.name}"?`,
			async () => {
				try {
					const { removeWorktree } = await import("../lib/git.js");
					await removeWorktree(worktree.path);
					// Refresh worktrees after deletion
					const { listWorktrees } = await import("../lib/git.js");
					const updatedWorktrees = await listWorktrees();
					setWorktrees(updatedWorktrees);
					setFilteredWorktrees(updatedWorktrees);
					// Reset selection if needed
					if (
						modalState.state.selectedWorktreeIndex >= updatedWorktrees.length
					) {
						modalState.setSelectedIndex(
							Math.max(0, updatedWorktrees.length - 1),
						);
					}
				} catch (error) {
					console.error("Failed to delete worktree:", error);
				}
				modalState.hideConfirm();
			},
			() => modalState.hideConfirm(),
		);
	};

	const handleRenameWorktree = (worktree: Worktree) => {
		// TODO: Implement rename worktree modal/prompt
		modalState.showConfirm(
			"Rename worktree feature not implemented yet",
			() => modalState.hideConfirm(),
			() => modalState.hideConfirm(),
		);
	};

	// Handle keyboard input based on current mode.
	useInput((input, key) => {
		if (modalState.state.mode === "filter") {
			// Filter mode - handled by FilterModal.
			return;
		}

		if (modalState.state.mode === "help") {
			// Help mode - close on ? or escape.
			if (input === "?" || key.escape) {
				modalState.setMode("normal");
			}
			return;
		}

		if (modalState.state.mode === "confirm") {
			// Confirm mode - handled by ConfirmModal.
			return;
		}

		// Handle special keys first
		if (key.return) {
			// Switch to selected worktree
			if (selectedWorktree) {
				handleSwitchWorktree(selectedWorktree);
			}
			return;
		}

		if (key.escape) {
			if (modalState.state.mode !== "normal") {
				modalState.setMode("normal");
			}
			return;
		}

		// Handle arrow keys
		if (key.downArrow || input === "j") {
			handleNavigateDown();
			return;
		}

		if (key.upArrow || input === "k") {
			handleNavigateUp();
			return;
		}

		// Normal mode keybindings.
		switch (input) {
			case "q":
				onExit?.();
				process.exit(0);
				break;
			case "/":
				modalState.setMode("filter");
				break;
			case "?":
				modalState.setMode("help");
				break;
			case "c":
				handleCreateWorktree();
				break;
			case "d":
				if (selectedWorktree) {
					handleDeleteWorktree(selectedWorktree);
				}
				break;
			case "r":
				if (selectedWorktree) {
					handleRenameWorktree(selectedWorktree);
				}
				break;
		}
	});

	const selectedWorktree =
		filteredWorktrees[modalState.state.selectedWorktreeIndex];

	return (
		<Box flexDirection="column" height="100%">
			<HeaderBar worktrees={worktrees} />
			<Box flexGrow={1} flexDirection="row">
				<WorktreeListPanel
					worktrees={filteredWorktrees}
					selectedIndex={modalState.state.selectedWorktreeIndex}
					mode={modalState.state.mode}
				/>
				<DetailsPanel worktree={selectedWorktree} />
			</Box>
			<StatusLine
				mode={modalState.state.mode}
				worktreeCount={filteredWorktrees.length}
			/>

			{modalState.state.mode === "filter" && (
				<FilterModal
					query={modalState.state.searchQuery}
					onQueryChange={modalState.setSearchQuery}
					onApply={() => {
						modalState.setMode("normal");
					}}
					onCancel={() => {
						modalState.setSearchQuery("");
						modalState.setMode("normal");
					}}
					allWorktrees={worktrees}
				/>
			)}

			{modalState.state.mode === "help" && <HelpModal />}

			{modalState.state.mode === "confirm" &&
				modalState.state.confirmAction && (
					<ConfirmModal
						message={modalState.state.confirmAction.message}
						onConfirm={modalState.state.confirmAction.onConfirm}
						onCancel={modalState.state.confirmAction.onCancel}
					/>
				)}
		</Box>
	);
}
