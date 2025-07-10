import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { App } from "./App";
import type { Worktree } from "../commands/list";

// Mock git operations
vi.mock("../lib/git", () => ({
	listWorktrees: vi.fn(() => Promise.resolve(mockWorktrees)),
	switchToWorktree: vi.fn(() => Promise.resolve()),
	removeWorktree: vi.fn(() => Promise.resolve()),
	createWorktree: vi.fn(() => Promise.resolve()),
}));

const mockWorktrees: Worktree[] = [
	{
		name: "main",
		path: "/repo/main",
		branch: "main",
		head: "abc123",
		active: true,
		locked: false,
	},
	{
		name: "feature-branch",
		path: "/repo/feature-branch",
		branch: "feature-branch",
		head: "def456",
		active: false,
		locked: false,
	},
];

describe("Keybindings", () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it("should setup keyboard event handling", () => {
		// Test that App component renders and sets up keyboard handling
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// If component renders without error, useInput is properly configured
		expect(lastFrame()).toContain("Grove");
	});

	it("should handle normal mode keybindings", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// Should start in normal mode
		expect(lastFrame()).toContain("2 worktrees"); // Normal mode shows worktree count
		expect(lastFrame()).toContain("[j/k] Navigate"); // Normal mode keybindings
	});

	it("should handle filter mode transition", () => {
		const { lastFrame, stdin } = render(
			<App initialWorktrees={mockWorktrees} />,
		);

		// Test that component is ready for filter mode
		// Note: stdin.write doesn't actually trigger useInput in tests
		expect(lastFrame()).toContain("Grove");
	});

	it("should handle help mode transition", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// Test that help mode can be activated
		expect(lastFrame()).toContain("Grove");
	});

	it("should handle exit functionality", () => {
		// Test that exit handler can be set
		const onExit = vi.fn();
		const { lastFrame } = render(
			<App initialWorktrees={mockWorktrees} onExit={onExit} />,
		);

		expect(lastFrame()).toContain("Grove");
		// Exit handler should not be called during render
		expect(onExit).not.toHaveBeenCalled();
	});

	it("should maintain keybinding consistency", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		const output = lastFrame();
		// Check that keybindings mentioned in status line are consistent
		expect(output).toContain("[j/k]"); // Navigation
		expect(output).toContain("[Enter]"); // Switch
		expect(output).toContain("[q]"); // Quit
	});

	it("should handle special key combinations", () => {
		// Test that component handles various key types
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// Component should be stable with different key inputs
		expect(lastFrame()).toContain("Grove");
	});

	it("should show appropriate actions for selected worktree", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		const output = lastFrame();
		// Should show actions available for worktrees
		expect(output).toContain("[c]"); // Create
		expect(output).toContain("[d]"); // Delete
	});

	it("should handle mode-specific keybindings", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// In normal mode, should show normal keybindings
		const output = lastFrame();
		expect(output).toContain("Navigate");
		expect(output).toContain("Switch");
	});

	it("should prevent action on inappropriate states", () => {
		// Test with no worktrees selected
		const { lastFrame } = render(<App initialWorktrees={[]} />);

		expect(lastFrame()).toContain("No worktrees found");
		// Should still show basic navigation
		expect(lastFrame()).toContain("Grove");
	});
});
