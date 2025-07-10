import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect } from "vitest";
import { WorktreeListPanel } from "./WorktreeListPanel";
import type { Worktree } from "../commands/list";

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
		name: "locked-branch",
		path: "/repo/locked-branch",
		branch: "locked-branch",
		head: "ghi789",
		active: false,
		locked: true,
	},
];

describe("WorktreeListPanel", () => {
	it("should render worktree list with title", () => {
		const { lastFrame } = render(
			<WorktreeListPanel
				worktrees={mockWorktrees}
				selectedIndex={0}
				mode="normal"
			/>,
		);

		expect(lastFrame()).toContain("Worktrees");
		expect(lastFrame()).toContain("main");
		expect(lastFrame()).toContain("feature-branch");
		expect(lastFrame()).toContain("locked-branch");
	});

	it("should highlight selected worktree in normal mode", () => {
		const { lastFrame } = render(
			<WorktreeListPanel
				worktrees={mockWorktrees}
				selectedIndex={1}
				mode="normal"
			/>,
		);

		const output = lastFrame();
		expect(output).toContain("> feature-branch"); // Selected item has >
	});

	it("should not highlight when not in normal mode", () => {
		const { lastFrame } = render(
			<WorktreeListPanel
				worktrees={mockWorktrees}
				selectedIndex={0}
				mode="filter"
			/>,
		);

		const output = lastFrame();
		// In filter mode, no selection highlighting
		expect(output).toContain("main");
		expect(output).not.toContain("> main");
	});

	it("should show correct status indicators", () => {
		const { lastFrame } = render(
			<WorktreeListPanel
				worktrees={mockWorktrees}
				selectedIndex={0}
				mode="normal"
			/>,
		);

		const output = lastFrame();
		expect(output).toContain("*active"); // Active worktree
		expect(output).toContain("locked"); // Locked worktree
	});

	it("should handle empty worktree list", () => {
		const { lastFrame } = render(
			<WorktreeListPanel worktrees={[]} selectedIndex={0} mode="normal" />,
		);

		expect(lastFrame()).toContain("No worktrees found");
	});

	it("should handle out-of-bounds selected index", () => {
		const { lastFrame } = render(
			<WorktreeListPanel
				worktrees={mockWorktrees}
				selectedIndex={99}
				mode="normal"
			/>,
		);

		// Should not crash and should render normally
		expect(lastFrame()).toContain("Worktrees");
		expect(lastFrame()).toContain("main");
	});

	it("should show proper selection indicators", () => {
		// Test first item selected
		const { lastFrame: frame1 } = render(
			<WorktreeListPanel
				worktrees={mockWorktrees}
				selectedIndex={0}
				mode="normal"
			/>,
		);
		expect(frame1()).toContain("> main");

		// Test second item selected
		const { lastFrame: frame2 } = render(
			<WorktreeListPanel
				worktrees={mockWorktrees}
				selectedIndex={1}
				mode="normal"
			/>,
		);
		expect(frame2()).toContain("> feature-branch");
	});
});
