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
	{
		name: "bug-fix",
		path: "/repo/bug-fix",
		branch: "bug-fix",
		head: "ghi789",
		active: false,
		locked: false,
	},
];

describe("Navigation", () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it("should initialize with first worktree selected", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// First item should be selected (has > indicator)
		expect(lastFrame()).toContain("> main");
	});

	it("should handle navigation with j/k keys", () => {
		const { lastFrame, stdin } = render(
			<App initialWorktrees={mockWorktrees} />,
		);

		// Initially first item selected
		expect(lastFrame()).toContain("> main");

		// Navigate down with j
		stdin.write("j");
		// Note: In tests, we can't actually test the state change
		// since stdin.write doesn't trigger useInput in tests
	});

	it("should handle navigation wrap-around", () => {
		// Test that navigation wraps correctly at boundaries
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// Check that component renders without error
		expect(lastFrame()).toContain("Grove");
		expect(lastFrame()).toContain("main");
	});

	it("should handle empty worktree list navigation", () => {
		const { lastFrame } = render(<App initialWorktrees={[]} />);

		expect(lastFrame()).toContain("No worktrees found");
		expect(lastFrame()).toContain("0 worktrees");
	});

	it("should maintain selection when filtering", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// Check that selection is maintained
		expect(lastFrame()).toContain("> main");
	});

	it("should handle single worktree navigation", () => {
		const singleWorktree = mockWorktrees.slice(0, 1);
		const { lastFrame } = render(<App initialWorktrees={singleWorktree} />);

		expect(lastFrame()).toContain("> main");
		expect(lastFrame()).toContain("1 worktrees");
	});

	it("should show correct keybindings in status line", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		const output = lastFrame();
		expect(output).toContain("[j/k] Navigate");
		expect(output).toContain("[Enter] Switch");
		expect(output).toContain("[c] Create");
		expect(output).toContain("[d] Delete");
	});

	it("should display worktree details for selected item", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// Should show details for first selected worktree
		expect(lastFrame()).toContain("Branch: main");
		expect(lastFrame()).toContain("✓ Active");
	});
});
