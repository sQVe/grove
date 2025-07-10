import React from "react";
import { render } from "ink-testing-library";
import { describe, it, expect } from "vitest";
import { DetailsPanel } from "./DetailsPanel";
import type { Worktree } from "../commands/list";

const mockWorktree: Worktree = {
	name: "main",
	path: "/repo/main",
	branch: "main",
	head: "abc123def456",
	active: true,
	locked: false,
};

const lockedWorktree: Worktree = {
	name: "locked-branch",
	path: "/repo/locked-branch",
	branch: "locked-branch",
	head: "ghi789jkl012",
	active: false,
	locked: true,
};

const inactiveWorktree: Worktree = {
	name: "feature",
	path: "/repo/feature",
	branch: "feature",
	head: "mno345pqr678",
	active: false,
	locked: false,
};

describe("DetailsPanel", () => {
	it("should render worktree details", () => {
		const { lastFrame } = render(<DetailsPanel worktree={mockWorktree} />);

		const output = lastFrame();
		expect(output).toContain("Details");
		expect(output).toContain("Branch: main");
		expect(output).toContain("Path: /repo/main");
		expect(output).toContain("✓ Active");
		expect(output).toContain("abc123de"); // Truncated head
	});

	it("should show active status correctly", () => {
		const { lastFrame } = render(<DetailsPanel worktree={mockWorktree} />);

		expect(lastFrame()).toContain("✓ Active");
	});

	it("should show locked status correctly", () => {
		const { lastFrame } = render(<DetailsPanel worktree={lockedWorktree} />);

		expect(lastFrame()).toContain("🔒 Locked");
	});

	it("should show inactive status correctly", () => {
		const { lastFrame } = render(<DetailsPanel worktree={inactiveWorktree} />);

		expect(lastFrame()).toContain("◯ Inactive");
	});

	it("should handle no worktree selected", () => {
		const { lastFrame } = render(<DetailsPanel />);

		const output = lastFrame();
		expect(output).toContain("Details");
		expect(output).toContain("No worktree selected");
	});

	it("should truncate long commit hash", () => {
		const { lastFrame } = render(<DetailsPanel worktree={mockWorktree} />);

		const output = lastFrame();
		// Should show first 8 characters
		expect(output).toContain("abc123de");
		expect(output).not.toContain("abc123def456"); // Full hash should not appear
	});

	it("should display all required fields", () => {
		const { lastFrame } = render(<DetailsPanel worktree={mockWorktree} />);

		const output = lastFrame();
		expect(output).toContain("Branch:");
		expect(output).toContain("Path:");
		expect(output).toContain("Status:");
		expect(output).toContain("Head:");
	});

	it("should handle undefined worktree gracefully", () => {
		const { lastFrame } = render(<DetailsPanel worktree={undefined} />);

		expect(lastFrame()).toContain("No worktree selected");
	});
});
