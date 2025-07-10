import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { App } from "./App";
import type { Worktree } from "../commands/list";

// Mock the git operations
vi.mock("../lib/git", () => ({
	listWorktrees: vi.fn(() => Promise.resolve(mockWorktrees)),
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

describe("App", () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it("should render with initial worktrees", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		expect(lastFrame()).toContain("Grove");
		expect(lastFrame()).toContain("main");
		expect(lastFrame()).toContain("feature-branch");
	});

	it("should display correct worktree count", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		expect(lastFrame()).toContain("2 worktrees");
	});

	it("should show normal mode by default", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		expect(lastFrame()).toContain("2 worktrees"); // Normal mode shows worktree count without mode indicator
	});

	it("should handle empty worktree list", () => {
		const { lastFrame } = render(<App initialWorktrees={[]} />);

		expect(lastFrame()).toContain("No worktrees found");
		expect(lastFrame()).toContain("0 worktrees");
	});

	it("should show selected worktree with highlight", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// First item should be selected by default
		expect(lastFrame()).toContain("> main");
	});

	it("should display active worktree status", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		expect(lastFrame()).toContain("*active");
	});

	it("should show worktree details for selected item", () => {
		const { lastFrame } = render(<App initialWorktrees={mockWorktrees} />);

		// Should show details for first selected worktree (main)
		expect(lastFrame()).toContain("Branch: main");
		expect(lastFrame()).toContain("✓ Active");
		expect(lastFrame()).toContain("abc123");
	});
});
